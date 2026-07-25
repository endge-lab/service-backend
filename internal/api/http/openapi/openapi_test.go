package openapi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestGeneratedOpenAPISpecMatchesCheckedInContract(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "..", "docs", "openapi3.yaml")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read checked-in OpenAPI contract: %v", err)
	}

	if !bytes.Equal(openAPI3YAML, contract) {
		t.Fatal("generated OpenAPI bytes are stale; run make docs")
	}
}

func TestScopedOperationsRequireWorkspaceHeader(t *testing.T) {
	var document struct {
		Paths map[string]map[string]struct {
			Security   []map[string][]string `yaml:"security"`
			Parameters []struct {
				Name     string `yaml:"name"`
				In       string `yaml:"in"`
				Required bool   `yaml:"required"`
				Example  string `yaml:"example"`
			} `yaml:"parameters"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(openAPI3YAML, &document); err != nil {
		t.Fatalf("parse embedded OpenAPI document: %v", err)
	}

	checked := 0
	for path, operations := range document.Paths {
		if !strings.HasPrefix(path, "/api/v1/projects") {
			continue
		}
		for method, operation := range operations {
			if !isHTTPMethod(method) {
				continue
			}
			checked++
			if !requiresWorkspaceSecurity(operation.Security) {
				t.Errorf("%s %s does not require WorkspaceAuth", strings.ToUpper(method), path)
			}
			if !hasWorkspaceHeaderParameter(operation.Parameters) {
				t.Errorf("%s %s does not expose X-Endge-Workspace with demo-workspace example", strings.ToUpper(method), path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no workspace-scoped OpenAPI operations found")
	}
}

func requiresWorkspaceSecurity(requirements []map[string][]string) bool {
	for _, requirement := range requirements {
		if _, ok := requirement["WorkspaceAuth"]; ok {
			return true
		}
	}
	return false
}

func hasWorkspaceHeaderParameter(parameters []struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
	Example  string `yaml:"example"`
}) bool {
	for _, parameter := range parameters {
		if parameter.Name == "X-Endge-Workspace" &&
			parameter.In == "header" &&
			parameter.Required &&
			parameter.Example == "demo-workspace" {
			return true
		}
	}
	return false
}

func isHTTPMethod(value string) bool {
	switch value {
	case "get", "post", "put", "patch", "delete", "head", "options", "trace":
		return true
	default:
		return false
	}
}
