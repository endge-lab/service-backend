package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const accessTestFile = `version: 1
adapter: oidc
provider: primary
displayName: Test OIDC
claimsSource: access_token
rules:
  - id: viewer
    when: {path: /roles, contains: viewer}
    grant: {scope: workspace, workspace: operations, role: viewer}
  - id: editor
    when: {path: /abac/edit, equals: true}
    grant: {scope: workspace, workspace: operations, role: editor}
  - id: admin
    when: {path: /roles, contains: admin}
    grant: {scope: workspace, workspace: operations, role: admin}
  - id: platform
    when: {path: /roles, contains: platform-admin}
    grant: {scope: platform, role: admin}
`

func accessTestIdentity() IdentityConfig      { return IdentityConfig{Mode: "oidc", ProviderID: "primary"} }
func accessTestLogin() ConfiguratorAuthConfig { return ConfiguratorAuthConfig{Adapter: "oidc"} }

func TestAccessConfigStrictSchema(t *testing.T) {
	cases := map[string]string{
		"empty": "", "null document": "null", "empty object": "{}",
		"fractional version":      strings.Replace(accessTestFile, "version: 1", "version: 1.5", 1),
		"numeric contains":        strings.Replace(accessTestFile, "contains: viewer", "contains: 123", 1),
		"numeric displayName":     strings.Replace(accessTestFile, "displayName: Test OIDC", "displayName: 123", 1),
		"unknown field":           accessTestFile + "unknown: true\n",
		"duplicate key":           accessTestFile + "version: 1\n",
		"second document":         accessTestFile + "---\nversion: 1\n",
		"empty second document":   accessTestFile + "---\n",
		"version":                 strings.Replace(accessTestFile, "version: 1", "version: 2", 1),
		"adapter":                 strings.Replace(accessTestFile, "adapter: oidc", "adapter: bearer", 1),
		"provider":                strings.Replace(accessTestFile, "provider: primary", "provider: another", 1),
		"claims":                  strings.Replace(accessTestFile, "access_token", "id_token", 1),
		"duplicate rule":          strings.Replace(accessTestFile, "id: editor", "id: viewer", 1),
		"unknown nested":          strings.Replace(accessTestFile, "path: /roles", "extra: true, path: /roles", 1),
		"no operator":             strings.Replace(accessTestFile, ", contains: viewer", "", 1),
		"two operators":           strings.Replace(accessTestFile, "contains: viewer", "contains: viewer, equals: true", 1),
		"contains null":           strings.Replace(accessTestFile, "contains: viewer", "contains: null, equals: true", 1),
		"equals null":             strings.Replace(accessTestFile, "equals: true", "equals: null", 1),
		"equals object":           strings.Replace(accessTestFile, "equals: true", "equals: {x: true}", 1),
		"equals infinite":         strings.Replace(accessTestFile, "equals: true", "equals: .inf", 1),
		"pointer":                 strings.Replace(accessTestFile, "/abac/edit", "abac.edit", 1),
		"pointer escape":          strings.Replace(accessTestFile, "/abac/edit", "/abac/~2", 1),
		"platform viewer":         strings.Replace(accessTestFile, "scope: platform, role: admin", "scope: platform, role: viewer", 1),
		"platform workspace":      strings.Replace(accessTestFile, "scope: platform, role: admin", "scope: platform, workspace: operations, role: admin", 1),
		"platform null workspace": strings.Replace(accessTestFile, "scope: platform, role: admin", "scope: platform, workspace: null, role: admin", 1),
		"workspace missing":       strings.Replace(accessTestFile, "workspace: operations, ", "", 1),
		"wildcard workspace":      strings.Replace(accessTestFile, "workspace: operations", "workspace: '*'", 1),
		"invalid role":            strings.Replace(accessTestFile, "role: editor", "role: owner", 1),
		"alias":                   strings.Replace(accessTestFile, "when: {path: /roles, contains: viewer}", "when: &condition {path: /roles, contains: viewer}", 1) + "extra: *condition\n",
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseAccessConfig([]byte(data), accessTestIdentity(), accessTestLogin()); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	for _, data := range []string{accessTestFile, "version: 1\nadapter: oidc\nprovider: primary\ndisplayName: OIDC\nclaimsSource: access_token\nrules: []\n"} {
		if _, err := ParseAccessConfig([]byte(data), accessTestIdentity(), accessTestLogin()); err != nil {
			t.Fatal(err)
		}
	}
	dev := accessTestIdentity()
	dev.Mode = "dev"
	if _, err := ParseAccessConfig([]byte(accessTestFile), dev, accessTestLogin()); err == nil {
		t.Fatal("dev accepted external access")
	}
}

func TestAccessConfigMapsAllRolesAndStrictClaimTypes(t *testing.T) {
	policy, err := ParseAccessConfig([]byte(accessTestFile), accessTestIdentity(), accessTestLogin())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		json  string
		roles []string
	}{
		{`{}`, nil}, {`{"roles":"viewer"}`, nil}, {`{"roles":["viewer",false]}`, nil},
		{`{"abac":{"edit":"true"}}`, nil}, {`{"roles":["not-viewer"]}`, nil},
		{`{"roles":["viewer"]}`, []string{"workspace:viewer"}},
		{`{"roles":["viewer"],"abac":{"edit":true}}`, []string{"workspace:editor"}},
		{`{"roles":["viewer","admin"],"abac":{"edit":true}}`, []string{"workspace:admin"}},
		{`{"roles":["platform-admin","viewer"]}`, []string{"platform:admin", "workspace:viewer"}},
	}
	for _, tc := range cases {
		var claims map[string]any
		if err := json.Unmarshal([]byte(tc.json), &claims); err != nil {
			t.Fatal(err)
		}
		got := policy.Map(claims)
		if len(got) != len(tc.roles) {
			t.Fatalf("%s: grants=%v", tc.json, got)
		}
		for i, role := range tc.roles {
			if got[i].Scope+":"+got[i].Role != role {
				t.Fatalf("%s: grants=%v", tc.json, got)
			}
		}
	}
}

func TestAccessConfigFileLoading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endge-access.yaml")
	if policy, err := readAccessConfig(path, false, accessTestIdentity(), accessTestLogin()); err != nil || policy != nil {
		t.Fatalf("missing default: policy=%v err=%v", policy, err)
	}
	if _, err := readAccessConfig(path, true, accessTestIdentity(), accessTestLogin()); err == nil {
		t.Fatal("missing explicit path accepted")
	}
	if err := os.Symlink(path+"-missing", path); err != nil {
		t.Fatal(err)
	}
	if _, err := readAccessConfig(path, false, accessTestIdentity(), accessTestLogin()); err == nil {
		t.Fatal("broken default symlink became local mode")
	}
	t.Setenv("ACCESS_CONFIG_FILE", "relative.yaml")
	if _, err := loadAccessConfig(accessTestIdentity(), accessTestLogin()); err == nil {
		t.Fatal("relative path accepted")
	}
	explicit := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(explicit, []byte(accessTestFile), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ACCESS_CONFIG_FILE", explicit)
	policy, err := loadAccessConfig(accessTestIdentity(), accessTestLogin())
	if err != nil || !policy.External() {
		t.Fatalf("explicit config: %v", err)
	}
	if err := os.WriteFile(explicit, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if policy.Management().SourceName != "Test OIDC" {
		t.Fatal("loaded policy changed after file write")
	}
}

func TestExternalAccessExample(t *testing.T) {
	data, err := os.ReadFile("../../endge-access.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ParseAccessConfig(data, accessTestIdentity(), accessTestLogin())
	if err != nil {
		t.Fatal(err)
	}
	got := policy.Map(map[string]any{"resource_access": map[string]any{"endge-configurator": map[string]any{"roles": []any{"platform-admin", "operations-admin", "operations-viewer"}}}})
	want := []entities.MappedAccessGrant{{Scope: "platform", Role: "admin"}, {Scope: "workspace", Workspace: "operations", Role: "admin"}}
	left, _ := json.Marshal(got)
	right, _ := json.Marshal(want)
	if string(left) != string(right) {
		t.Fatalf("example grants=%s", left)
	}
}
