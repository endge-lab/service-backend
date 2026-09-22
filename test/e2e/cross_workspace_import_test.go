//go:build e2e

package e2e_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

func TestCrossWorkspaceImportPreservesFacetDocumentCounts(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	configuration := support.DevConfig()
	configuration.WorkspaceSchemaVersion = 9
	app := support.NewTestApp(t, database, configuration)
	platformHeaders := map[string]string{"X-Endge-Workspace": "default"}
	createdSource := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{
		"identity": "import-source", "displayName": "Import source", "dataMode": "development",
	}, platformHeaders)
	assertStatus(t, createdSource, fiber.StatusCreated)
	createdSource.Body.Close()
	sourceHeaders := map[string]string{"X-Endge-Workspace": "import-source"}

	facet := perform(t, app, http.MethodPost, "/api/v1/facets", map[string]any{
		"identity": "release-scope", "displayName": "Release scope", "icon": "Package", "color": "#2563eb",
	}, sourceHeaders)
	assertStatus(t, facet, fiber.StatusCreated)
	facet.Body.Close()
	document := perform(t, app, http.MethodPost, "/api/v1/facets/release-scope/documents", map[string]any{
		"identity": "production", "displayName": "Production", "configuration": map[string]any{"mode": "inherit", "patch": map[string]any{}},
	}, sourceHeaders)
	assertStatus(t, document, fiber.StatusCreated)
	document.Body.Close()
	commit := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{
		"message": "Cross-workspace import source", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, sourceHeaders),
	}, sourceHeaders)
	assertStatus(t, commit, fiber.StatusCreated)
	commit.Body.Close()
	exported := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, sourceHeaders))

	createdTarget := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{
		"identity": "import-target", "displayName": "Import target", "dataMode": "development",
	}, platformHeaders)
	assertStatus(t, createdTarget, fiber.StatusCreated)
	createdTarget.Body.Close()
	targetHeaders := map[string]string{"X-Endge-Workspace": "import-target"}
	planResponse := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", exported, targetHeaders)
	assertStatus(t, planResponse, fiber.StatusOK)
	plan := decodeObject(t, planResponse)
	if valid, _ := plan["valid"].(bool); !valid {
		t.Fatalf("cross-workspace import plan is invalid: %#v", plan)
	}
	importHeaders := cloneHeaders(targetHeaders)
	importHeaders["If-Match"] = stringField(t, plan, "targetETag")
	imported := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, plan, "planId"), "confirmation": "import-target",
	}, importHeaders)
	assertStatus(t, imported, fiber.StatusCreated)
	if importedVersion := stringField(t, decodeObject(t, imported), "domainVersion"); importedVersion != stringField(t, exported, "domainVersion") {
		t.Fatalf("cross-workspace import changed domainVersion: export=%q import=%q", stringField(t, exported, "domainVersion"), importedVersion)
	}
}

func TestCrossWorkspaceImportRetainsTargetOnlyFacetTombstones(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	configuration := support.DevConfig()
	configuration.WorkspaceSchemaVersion = 9
	app := support.NewTestApp(t, database, configuration)
	platformHeaders := map[string]string{"X-Endge-Workspace": "default"}

	for _, identity := range []string{"tombstone-source", "tombstone-target"} {
		response := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{
			"identity": identity, "displayName": identity, "dataMode": "development",
		}, platformHeaders)
		assertStatus(t, response, fiber.StatusCreated)
		response.Body.Close()
	}
	sourceHeaders := map[string]string{"X-Endge-Workspace": "tombstone-source"}
	targetHeaders := map[string]string{"X-Endge-Workspace": "tombstone-target"}

	createFacet := func(headers map[string]string, identity string) {
		facet := perform(t, app, http.MethodPost, "/api/v1/facets", map[string]any{
			"identity": identity, "displayName": identity, "icon": "Package", "color": "#2563eb",
		}, headers)
		assertStatus(t, facet, fiber.StatusCreated)
		facet.Body.Close()
		document := perform(t, app, http.MethodPost, "/api/v1/facets/"+identity+"/documents", map[string]any{
			"identity": "value", "displayName": "Value", "configuration": map[string]any{"mode": "inherit", "patch": map[string]any{}},
		}, headers)
		assertStatus(t, document, fiber.StatusCreated)
		document.Body.Close()
		commit := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{
			"message": "Facet state", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, headers),
		}, headers)
		assertStatus(t, commit, fiber.StatusCreated)
		commit.Body.Close()
	}
	createFacet(sourceHeaders, "source-scope")
	createFacet(targetHeaders, "target-scope")

	exported := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, sourceHeaders))
	planResponse := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", exported, targetHeaders)
	assertStatus(t, planResponse, fiber.StatusOK)
	plan := decodeObject(t, planResponse)
	if valid, _ := plan["valid"].(bool); !valid {
		t.Fatalf("cross-workspace import plan is invalid: %#v", plan)
	}
	warnings, _ := plan["warnings"].([]any)
	foundWarning := false
	for _, warning := range warnings {
		if strings.Contains(warning.(string), "Target-only Facet tombstones will be retained: 2") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("target-only tombstone warning is missing: %#v", warnings)
	}
	importHeaders := cloneHeaders(targetHeaders)
	importHeaders["If-Match"] = stringField(t, plan, "targetETag")
	imported := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, plan, "planId"), "confirmation": "tombstone-target",
	}, importHeaders)
	assertStatus(t, imported, fiber.StatusCreated)
	importedBody := decodeObject(t, imported)
	roundTrip := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, targetHeaders))
	if stringField(t, importedBody, "domainVersion") != stringField(t, roundTrip, "domainVersion") {
		t.Fatalf("imported domainVersion does not match persisted export: import=%q export=%q", stringField(t, importedBody, "domainVersion"), stringField(t, roundTrip, "domainVersion"))
	}
}
