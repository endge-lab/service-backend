package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/access"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"go.yaml.in/yaml/v3"
)

const accessConfigName = "endge-access.yaml"

type accessFile struct {
	Version      int           `yaml:"version"`
	Adapter      string        `yaml:"adapter"`
	Provider     string        `yaml:"provider"`
	DisplayName  string        `yaml:"displayName"`
	ClaimsSource string        `yaml:"claimsSource"`
	Rules        *[]accessRule `yaml:"rules"`
}

type accessRule struct {
	ID   string `yaml:"id"`
	When struct {
		Path     string    `yaml:"path"`
		Equals   yaml.Node `yaml:"equals"`
		Contains *string   `yaml:"contains"`
	} `yaml:"when"`
	Grant struct {
		Scope     string  `yaml:"scope"`
		Workspace *string `yaml:"workspace"`
		Role      string  `yaml:"role"`
	} `yaml:"grant"`
}

func loadAccessConfig(identity IdentityConfig, login ConfiguratorAuthConfig) (*access.Policy, error) {
	path := strings.TrimSpace(os.Getenv("ACCESS_CONFIG_FILE"))
	explicit := path != ""
	if explicit && !filepath.IsAbs(path) {
		return nil, fmt.Errorf("ACCESS_CONFIG_FILE must be an absolute path")
	}
	if !explicit {
		executable, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve access config directory: %w", err)
		}
		path = filepath.Join(filepath.Dir(executable), accessConfigName)
	}
	return readAccessConfig(path, explicit, identity, login)
}

func readAccessConfig(path string, explicit bool, identity IdentityConfig, login ConfiguratorAuthConfig) (*access.Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !explicit && os.IsNotExist(err) {
			if _, statErr := os.Lstat(path); os.IsNotExist(statErr) {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("read access configuration: %w", err)
	}
	return ParseAccessConfig(data, identity, login)
}

// ParseAccessConfig validates and compiles one complete startup configuration.
func ParseAccessConfig(data []byte, identity IdentityConfig, login ConfiguratorAuthConfig) (*access.Policy, error) {
	if identity.Mode != IdentityModeOIDC || login.Adapter != IdentityModeOIDC {
		return nil, fmt.Errorf("external access requires OIDC identity and login")
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode access configuration: %w", err)
	}
	if err := validateAccessYAMLNode(&document); err != nil {
		return nil, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var file accessFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode access configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("access configuration must contain exactly one YAML document")
	}
	if file.Version != 1 || file.Adapter != "oidc" || file.ClaimsSource != "access_token" {
		return nil, fmt.Errorf("access configuration requires version 1, adapter oidc and claimsSource access_token")
	}
	if strings.TrimSpace(file.Provider) == "" || file.Provider != identity.ProviderID || strings.TrimSpace(file.DisplayName) == "" || file.Rules == nil {
		return nil, fmt.Errorf("access configuration requires provider matching AUTH_PROVIDER_ID, displayName and rules")
	}
	ids := map[string]bool{}
	rules := make([]access.Rule, 0, len(*file.Rules))
	for _, input := range *file.Rules {
		if strings.TrimSpace(input.ID) == "" || ids[input.ID] {
			return nil, fmt.Errorf("access rule id must be nonempty and unique")
		}
		ids[input.ID] = true
		path, ok := access.ParsePointer(input.When.Path)
		if !ok {
			return nil, fmt.Errorf("access rule %q has invalid JSON Pointer", input.ID)
		}
		if (input.When.Equals.Kind != 0) == (input.When.Contains != nil) {
			return nil, fmt.Errorf("access rule %q requires exactly one of equals or contains", input.ID)
		}
		rule := access.Rule{Path: path, Contains: input.When.Contains, Grant: entities.MappedAccessGrant{Scope: input.Grant.Scope, Role: input.Grant.Role}}
		if input.When.Contains == nil {
			node := input.When.Equals
			if node.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("access rule %q equals must be scalar", input.ID)
			}
			switch node.Tag {
			case "!!str":
				rule.Equals = node.Value
			case "!!bool":
				var value bool
				if err := node.Decode(&value); err != nil {
					return nil, err
				}
				rule.Equals = value
			case "!!int", "!!float":
				var value any
				if err := node.Decode(&value); err != nil {
					return nil, err
				}
				raw, err := json.Marshal(value)
				if err != nil {
					return nil, fmt.Errorf("access rule %q requires a finite number", input.ID)
				}
				rule.Equals = json.Number(raw)
			default:
				return nil, fmt.Errorf("access rule %q equals must be string, boolean or number", input.ID)
			}
		}
		switch input.Grant.Scope {
		case "platform":
			if input.Grant.Role != "admin" || input.Grant.Workspace != nil {
				return nil, fmt.Errorf("access rule %q platform grant only accepts admin without workspace", input.ID)
			}
		case "workspace":
			if input.Grant.Workspace == nil || strings.TrimSpace(*input.Grant.Workspace) == "" || *input.Grant.Workspace == "*" {
				return nil, fmt.Errorf("access rule %q requires a fixed workspace identity", input.ID)
			}
			if input.Grant.Role != "viewer" && input.Grant.Role != "editor" && input.Grant.Role != "admin" {
				return nil, fmt.Errorf("access rule %q has invalid workspace role", input.ID)
			}
			rule.Grant.Workspace = *input.Grant.Workspace
		default:
			return nil, fmt.Errorf("access rule %q has invalid scope", input.ID)
		}
		rules = append(rules, rule)
	}
	// Include trust configuration: persisted sessions cannot reuse a mapping after its trust boundary changes.
	trust, _ := json.Marshal(struct {
		Identity IdentityConfig
		Login    ConfiguratorAuthConfig
	}{identity, login})
	hash := sha256.New()
	_, _ = hash.Write(data)
	_, _ = hash.Write(trust)
	return access.NewPolicy(file.Provider, file.DisplayName, hex.EncodeToString(hash.Sum(nil)), rules), nil
}

// Reject YAML-only indirection and nulls: every accepted field has explicit semantics.
func validateAccessYAMLNode(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Tag == "!!merge" || node.Tag == "!!null" {
		return fmt.Errorf("access configuration does not support aliases, merge keys or null values")
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if key.Tag != "!!str" {
				return fmt.Errorf("access configuration keys must be strings")
			}
			want := ""
			switch key.Value {
			case "version":
				want = "!!int"
			case "rules":
				want = "!!seq"
			case "when", "grant":
				want = "!!map"
			case "adapter", "provider", "displayName", "claimsSource", "id", "path", "contains", "scope", "workspace", "role":
				want = "!!str"
			}
			if want != "" && value.Tag != want {
				return fmt.Errorf("access configuration field %q has invalid type", key.Value)
			}
		}
	}
	for _, child := range node.Content {
		if err := validateAccessYAMLNode(child); err != nil {
			return err
		}
	}
	return nil
}
