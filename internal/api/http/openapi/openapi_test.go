package openapi

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestGeneratedOpenAPISpecMatchesCheckedInContract проверяет актуальность встроенной спецификации.
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

// TestEveryLocalOpenAPIReferenceResolves проверяет все локальные ссылки OpenAPI.
func TestEveryLocalOpenAPIReferenceResolves(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(openAPI3YAML, &document); err != nil {
		t.Fatalf("parse embedded OpenAPI document: %v", err)
	}
	walkOpenAPI(t, document, document)
}

func walkOpenAPI(t *testing.T, root map[string]any, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if key == "$ref" {
				reference, ok := nested.(string)
				if ok && strings.HasPrefix(reference, "#/") {
					resolveLocalReference(t, root, reference)
				}
			}
			walkOpenAPI(t, root, nested)
		}
	case []any:
		for _, nested := range typed {
			walkOpenAPI(t, root, nested)
		}
	}
}

func resolveLocalReference(t *testing.T, root map[string]any, reference string) {
	t.Helper()
	var current any = root
	for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI reference %q crosses a non-object at %q", reference, segment)
		}
		current, ok = object[segment]
		if !ok {
			t.Fatalf("OpenAPI reference %q does not resolve", reference)
		}
	}
}

// TestDocumentContractContainsOnlyMVPCollections проверяет публичный контракт согласованных коллекций.
func TestDocumentContractContainsOnlyMVPCollections(t *testing.T) {
	var document struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(openAPI3YAML, &document); err != nil {
		t.Fatalf("parse embedded OpenAPI document: %v", err)
	}
	collections := []struct {
		path        string
		packageName string
	}{
		{"projects", "project"}, {"tenants", "tenant"}, {"environments", "environment"},
		{"folders", "folder"}, {"types", "domain_type"}, {"queries", "query"},
		{"data-views", "data_view"}, {"compositions", "composition"}, {"stores", "store"},
		{"streams", "stream"}, {"updates", "update"}, {"mocks", "mock"},
		{"components", "component"}, {"actions", "action"}, {"filters", "filter"},
		{"converters", "converter"}, {"computations", "computation"}, {"vocabs", "vocab"},
		{"i18n-bundles", "i18n_bundle"}, {"auth-profiles", "auth_profile"},
		{"navigations", "navigation"}, {"styles", "style"},
	}
	for _, collection := range collections {
		base := "/api/v1/" + collection.path
		item := base + "/{identity}"
		restore := item + "/restore"
		assertOperations(t, document.Paths[base], base, "get", "post")
		assertOperations(t, document.Paths[item], item, "get", "patch", "delete")
		assertOperations(t, document.Paths[restore], restore, "post")
		for _, schema := range []string{collection.packageName + ".CreateRequest", collection.packageName + ".PatchRequest", collection.packageName + ".Response", collection.packageName + ".ListResponse"} {
			if _, ok := document.Components.Schemas[schema]; !ok {
				t.Fatalf("schema %q is missing", schema)
			}
		}
		for _, method := range []string{"post"} {
			assertOperationRequestBody(t, document.Paths[base][method], method+" "+base)
		}
		assertOperationRequestBody(t, document.Paths[item]["patch"], "patch "+item)
		listContract := fmt.Sprint(document.Paths[base]["get"])
		for _, parameter := range []string{"X-Endge-Workspace", "includeDeleted", "folderIdentity", "active", "limit", "offset"} {
			if !bytes.Contains([]byte(listContract), []byte(parameter)) {
				t.Fatalf("%s GET does not expose %s", base, parameter)
			}
		}
	}
	for _, generic := range []string{"/api/v1/{collection}", "/api/v1/{collection}/{identity}"} {
		if _, ok := document.Paths[generic]; ok {
			t.Fatalf("generic Scalar path %q must not be present", generic)
		}
	}
	for _, excluded := range []string{"parameters", "pages", "page-templates", "policies", "versions"} {
		if _, ok := document.Paths["/api/v1/"+excluded]; ok {
			t.Fatalf("excluded collection %q is present", excluded)
		}
	}
	assertSchemaProperty(t, document.Components.Schemas, "query.CreateRequest", "sourceVersion")
	assertSchemaProperty(t, document.Components.Schemas, "update.PatchRequest", "storeIdentity")
	assertGeneratedOperationMetadata(t, document.Paths)
}

func assertOperations(t *testing.T, pathItem map[string]any, path string, methods ...string) {
	t.Helper()
	if pathItem == nil {
		t.Fatalf("path %q is missing", path)
	}
	for _, method := range methods {
		if _, ok := pathItem[method]; !ok {
			t.Fatalf("%s %s is missing", method, path)
		}
	}
}

func assertOperationRequestBody(t *testing.T, operation any, label string) {
	t.Helper()
	value, ok := operation.(map[string]any)
	if !ok || value["requestBody"] == nil {
		t.Fatalf("%s has no generated request body", label)
	}
}

func assertSchemaProperty(t *testing.T, schemas map[string]any, schemaName, propertyName string) {
	t.Helper()
	schema, ok := schemas[schemaName].(map[string]any)
	if !ok {
		t.Fatalf("schema %q has invalid shape", schemaName)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema %q properties have invalid shape", schemaName)
	}
	if _, ok := properties[propertyName]; !ok {
		t.Fatalf("schema %q does not expose property %q", schemaName, propertyName)
	}
}

func assertGeneratedOperationMetadata(t *testing.T, paths map[string]map[string]any) {
	t.Helper()
	methods := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
	operationIDs := map[string]string{}
	for path, item := range paths {
		for method, raw := range item {
			if !methods[method] {
				continue
			}
			operation, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("%s %s has invalid operation shape", method, path)
			}
			for _, field := range []string{"operationId", "summary", "description", "responses"} {
				if operation[field] == nil || operation[field] == "" {
					t.Fatalf("%s %s has no generated %s", method, path, field)
				}
			}
			operationID := operation["operationId"].(string)
			if previous, exists := operationIDs[operationID]; exists {
				t.Fatalf("operationId %q is duplicated for %s and %s %s", operationID, previous, method, path)
			}
			operationIDs[operationID] = method + " " + path
		}
	}
}
