package contract_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestCriticalOpenAPIEndpoints проверяет наличие всех публичных MVP-групп в checked-in контракте.
func TestCriticalOpenAPIEndpoints(t *testing.T) {
	document := readOpenAPI(t)
	expected := map[string][]string{
		"/health": {"get"}, "/version": {"get"}, "/auth/login": {"get"}, "/auth/callback": {"get"}, "/auth/logout": {"post"},
		"/api/session/me": {"get"}, "/api/v1/workspaces": {"get", "post"},
		"/api/v1/workspaces/{identity}": {"get", "patch"}, "/api/v1/workspaces/{identity}/members": {"get"},
		"/api/v1/workspaces/{identity}/members/{userId}": {"put", "delete"},
		"/api/v1/integrations":                           {"get", "post"}, "/api/v1/commits/plan": {"post"}, "/api/v1/commits": {"get", "post"},
		"/api/v1/commits/{id}/restore/plan": {"post"}, "/api/v1/commits/{id}/restore": {"post"},
		"/api/v1/domain": {"get"}, "/api/v1/domain/export": {"get"}, "/api/v1/domain/import/plan": {"post"}, "/api/v1/domain/import": {"post"},
		"/api/v1/domain/backups": {"get", "post"}, "/api/v1/domain/backups/archive": {"get"}, "/api/v1/domain/backups/{id}/export": {"get"},
		"/api/v1/releases": {"get", "post"}, "/api/v1/releases/{identity}/export": {"get"},
		"/api/v1/releases/{identity}/restore/plan": {"post"}, "/api/v1/releases/{identity}/restore": {"post"},
		"/api/v1/domain/documents/{type}/{identity}/revisions":                      {"get"},
		"/api/v1/domain/documents/{type}/{identity}/revisions/{revisionId}/restore": {"post"},
	}
	for path, methods := range expected {
		item, ok := document.Paths[path]
		if !ok {
			t.Fatalf("OpenAPI path %s отсутствует", path)
		}
		for _, method := range methods {
			if _, ok = item[method]; !ok {
				t.Fatalf("OpenAPI operation %s %s отсутствует", method, path)
			}
		}
	}
}

// TestDocumentOpenAPIConcurrencyContract проверяет workspace header, If-Match и responses всех CRUD-коллекций.
func TestDocumentOpenAPIConcurrencyContract(t *testing.T) {
	document := readOpenAPI(t)
	collections := []string{"projects", "tenants", "environments", "folders", "types", "queries", "data-views", "compositions", "stores", "streams", "updates", "mocks", "components", "actions", "filters", "converters", "computations", "vocabs", "i18n-bundles", "auth-profiles", "navigations", "styles"}
	for _, collection := range collections {
		base := "/api/v1/" + collection
		item := base + "/{identity}"
		for _, operation := range []struct {
			path   string
			method string
		}{
			{path: base, method: "post"}, {path: base, method: "get"}, {path: item, method: "get"},
			{path: item, method: "patch"}, {path: item, method: "delete"}, {path: item + "/restore", method: "post"},
		} {
			contract := document.Paths[operation.path][operation.method]
			if contract == nil {
				t.Fatalf("нет %s %s", operation.method, operation.path)
			}
			if !hasParameter(contract, "X-Endge-Workspace", "header") {
				t.Fatalf("%s %s не содержит X-Endge-Workspace", operation.method, operation.path)
			}
			if operation.method == "patch" || operation.method == "delete" || operation.path == item+"/restore" {
				if !hasParameter(contract, "If-Match", "header") {
					t.Fatalf("%s %s не содержит If-Match", operation.method, operation.path)
				}
				responses, _ := contract["responses"].(map[string]any)
				if responses["428"] == nil || responses["409"] == nil {
					t.Fatalf("%s %s не документирует 428/409", operation.method, operation.path)
				}
			}
		}
	}
}

type openAPIDocument struct {
	Paths map[string]map[string]map[string]any `yaml:"paths"`
}

func readOpenAPI(t *testing.T) openAPIDocument {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("определить путь contract test")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "openapi3.yaml"))
	if err != nil {
		t.Fatalf("прочитать OpenAPI: %v", err)
	}
	var document openAPIDocument
	if err = yaml.Unmarshal(raw, &document); err != nil {
		t.Fatalf("разобрать OpenAPI: %v", err)
	}
	return document
}

func hasParameter(operation map[string]any, name, location string) bool {
	parameters, _ := operation["parameters"].([]any)
	for _, value := range parameters {
		parameter, _ := value.(map[string]any)
		if parameter["name"] == name && parameter["in"] == location {
			return true
		}
	}
	return false
}
