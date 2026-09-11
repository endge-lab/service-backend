//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

func TestFacetAuthoringContract(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	config := support.DevConfig()
	config.WorkspaceSchemaVersion = 6
	app := support.NewTestApp(t, database, config)
	headers := map[string]string{"X-Endge-Workspace": "default"}

	createFacet := func(identity, name, icon, color string, meta map[string]any) map[string]any {
		response := perform(t, app, http.MethodPost, "/api/v1/facets", map[string]any{
			"identity": identity, "displayName": name, "icon": icon, "color": color, "meta": meta,
		}, headers)
		assertStatus(t, response, fiber.StatusCreated)
		return decodeObject(t, response)
	}
	region := createFacet("region", "Region", "MapPin", "#2563eb", map[string]any{"user": map[string]any{"owner": "platform"}})
	brand := createFacet("brand", "Brand", "Tag", "#16a34a", map[string]any{})
	if numberField(t, region, "position") != 0 || numberField(t, brand, "position") != 1 {
		t.Fatalf("initial facet order is invalid: region=%#v brand=%#v", region, brand)
	}

	createDocument := func(facet, identity, name string) map[string]any {
		response := perform(t, app, http.MethodPost, "/api/v1/facets/"+facet+"/documents", map[string]any{
			"identity":      identity,
			"displayName":   name,
			"configuration": map[string]any{"mode": "inherit", "patch": map[string]any{"locale": facet}},
			"meta":          map[string]any{"user": map[string]any{"facet": facet}},
		}, headers)
		assertStatus(t, response, fiber.StatusCreated)
		return decodeObject(t, response)
	}
	regionDocument := createDocument("region", "default", "Region default")
	brandDocument := createDocument("brand", "default", "Brand default")
	if stringField(t, regionDocument, "id") == stringField(t, brandDocument, "id") {
		t.Fatal("same document identity in different facets resolved to one record")
	}

	lockedHeaders := cloneHeaders(headers)
	lockedHeaders["If-Match"] = `"1"`
	lockedRename := perform(t, app, http.MethodPatch, "/api/v1/facets/region", map[string]any{"identity": "region-next"}, lockedHeaders)
	assertStatus(t, lockedRename, fiber.StatusConflict)
	lockedRename.Body.Close()
	lockedDelete := perform(t, app, http.MethodDelete, "/api/v1/facets/region", nil, lockedHeaders)
	assertStatus(t, lockedDelete, fiber.StatusConflict)
	lockedDelete.Body.Close()

	presentation := perform(t, app, http.MethodPatch, "/api/v1/facets/region", map[string]any{
		"displayName": "Sales region", "icon": "Globe2", "color": "#7c3aed",
	}, lockedHeaders)
	assertStatus(t, presentation, fiber.StatusOK)
	if presentation.Header.Get("ETag") != `"2"` {
		t.Fatalf("facet presentation ETag=%q", presentation.Header.Get("ETag"))
	}
	presentation.Body.Close()

	reorder := perform(t, app, http.MethodPost, "/api/v1/facets/reorder", map[string]any{"items": []map[string]any{
		{"identity": "brand", "expectedRevision": 1},
		{"identity": "region", "expectedRevision": 2},
	}}, headers)
	assertStatus(t, reorder, fiber.StatusOK)
	reordered := decodeObject(t, reorder)
	items := arrayField(t, reordered, "items")
	if stringField(t, items[0].(map[string]any), "identity") != "brand" || stringField(t, items[1].(map[string]any), "identity") != "region" {
		t.Fatalf("unexpected facet order: %#v", items)
	}
	staleReorder := perform(t, app, http.MethodPost, "/api/v1/facets/reorder", map[string]any{"items": []map[string]any{
		{"identity": "region", "expectedRevision": 2},
		{"identity": "brand", "expectedRevision": 1},
	}}, headers)
	assertStatus(t, staleReorder, fiber.StatusConflict)
	staleReorder.Body.Close()

	facetRevisions := perform(t, app, http.MethodGet, "/api/v1/facets/region/revisions", nil, headers)
	assertStatus(t, facetRevisions, fiber.StatusOK)
	if len(arrayField(t, decodeObject(t, facetRevisions), "items")) < 3 {
		t.Fatal("facet reorder/presentation revisions were not recorded")
	}
	documentRevisions := perform(t, app, http.MethodGet, "/api/v1/facets/region/documents/default/revisions", nil, headers)
	assertStatus(t, documentRevisions, fiber.StatusOK)
	documentRevisionItems := arrayField(t, decodeObject(t, documentRevisions), "items")
	if len(documentRevisionItems) != 1 {
		t.Fatal("facet document create revision was not recorded")
	}
	firstDocumentRevisionID := stringField(t, documentRevisionItems[0].(map[string]any), "id")
	patchDocumentHeaders := cloneHeaders(headers)
	patchDocumentHeaders["If-Match"] = `"1"`
	patchedDocument := perform(t, app, http.MethodPatch, "/api/v1/facets/region/documents/default", map[string]any{"displayName": "Changed"}, patchDocumentHeaders)
	assertStatus(t, patchedDocument, fiber.StatusOK)
	patchedDocument.Body.Close()
	restoreRevisionHeaders := cloneHeaders(headers)
	restoreRevisionHeaders["If-Match"] = `"2"`
	restoredRevision := perform(t, app, http.MethodPost, "/api/v1/facets/region/documents/default/revisions/"+firstDocumentRevisionID+"/restore", nil, restoreRevisionHeaders)
	assertStatus(t, restoredRevision, fiber.StatusOK)
	if stringField(t, decodeObject(t, restoredRevision), "displayName") != "Region default" {
		t.Fatal("facet document revision restore did not restore the original snapshot")
	}

	createFacet("archive", "Archive", "Archive", "#ea580c", map[string]any{"user": map[string]any{"retention": "portable"}})
	createDocument("archive", "legacy", "Legacy settings")
	deleteArchiveDocumentHeaders := cloneHeaders(headers)
	deleteArchiveDocumentHeaders["If-Match"] = `"1"`
	deletedArchiveDocument := perform(t, app, http.MethodDelete, "/api/v1/facets/archive/documents/legacy", nil, deleteArchiveDocumentHeaders)
	assertStatus(t, deletedArchiveDocument, fiber.StatusOK)
	deletedArchiveDocument.Body.Close()
	deleteArchiveHeaders := cloneHeaders(headers)
	deleteArchiveHeaders["If-Match"] = `"1"`
	deletedArchive := perform(t, app, http.MethodDelete, "/api/v1/facets/archive", nil, deleteArchiveHeaders)
	assertStatus(t, deletedArchive, fiber.StatusOK)
	deletedArchive.Body.Close()

	commit := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{
		"message": "Facet authoring baseline", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, headers),
	}, headers)
	assertStatus(t, commit, fiber.StatusCreated)
	commit.Body.Close()

	exported := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, headers))
	if numberField(t, exported, "schemaVersion") != 6 {
		t.Fatalf("facet export schema=%v", exported["schemaVersion"])
	}
	documents := objectField(t, exported, "documents")
	exportedFacets := arrayField(t, documents, "facets")
	if len(exportedFacets) != 3 || stringField(t, exportedFacets[0].(map[string]any), "identity") != "brand" {
		t.Fatalf("portable facet order was not preserved: %#v", exportedFacets)
	}
	assertPortableFacetTombstone(t, exportedFacets, "", "archive")
	exportedDocuments := arrayField(t, documents, "facet-documents")
	if len(exportedDocuments) != 3 {
		t.Fatalf("portable facet documents=%#v", exportedDocuments)
	}
	assertPortableFacetTombstone(t, exportedDocuments, "archive", "legacy")

	planResponse := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", exported, headers)
	assertStatus(t, planResponse, fiber.StatusOK)
	plan := decodeObject(t, planResponse)
	if valid, _ := plan["valid"].(bool); !valid {
		t.Fatalf("facet export did not pass import plan: %#v", plan)
	}
	importHeaders := cloneHeaders(headers)
	importHeaders["If-Match"] = stringField(t, plan, "targetETag")
	imported := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, plan, "planId"), "confirmation": "default",
	}, importHeaders)
	assertStatus(t, imported, fiber.StatusCreated)
	imported.Body.Close()
	reExported := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, headers))
	reDocuments := objectField(t, reExported, "documents")
	if !reflect.DeepEqual(exportedFacets, arrayField(t, reDocuments, "facets")) || !reflect.DeepEqual(exportedDocuments, arrayField(t, reDocuments, "facet-documents")) {
		t.Fatalf("facet export/import/export changed portable content: before=%#v/%#v after=%#v/%#v", exportedFacets, exportedDocuments, reDocuments["facets"], reDocuments["facet-documents"])
	}

	deleteRegionDocumentHeaders := cloneHeaders(headers)
	regionDocumentAfterImport := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/facets/region/documents/default", nil, headers))
	regionDocumentRevision := int(numberField(t, regionDocumentAfterImport, "revision"))
	deleteRegionDocumentHeaders["If-Match"] = fmt.Sprintf(`"%d"`, regionDocumentRevision)
	deletedDocument := perform(t, app, http.MethodDelete, "/api/v1/facets/region/documents/default", nil, deleteRegionDocumentHeaders)
	assertStatus(t, deletedDocument, fiber.StatusOK)
	deletedDocument.Body.Close()
	deleteRegionHeaders := cloneHeaders(headers)
	regionAfterImportResponse := perform(t, app, http.MethodGet, "/api/v1/facets/region", nil, headers)
	assertStatus(t, regionAfterImportResponse, fiber.StatusOK)
	regionAfterImport := decodeObject(t, regionAfterImportResponse)
	deleteRegionHeaders["If-Match"] = fmt.Sprintf(`"%d"`, int(numberField(t, regionAfterImport, "revision")))
	deletedFacet := perform(t, app, http.MethodDelete, "/api/v1/facets/region", nil, deleteRegionHeaders)
	assertStatus(t, deletedFacet, fiber.StatusOK)
	deletedFacet.Body.Close()
	restoreDeletedDocumentHeaders := cloneHeaders(headers)
	restoreDeletedDocumentHeaders["If-Match"] = fmt.Sprintf(`"%d"`, regionDocumentRevision+1)
	blockedRestore := perform(t, app, http.MethodPost, "/api/v1/facets/region/documents/default/restore", nil, restoreDeletedDocumentHeaders)
	assertStatus(t, blockedRestore, fiber.StatusConflict)
	blockedRestore.Body.Close()

	live := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain", nil, headers))
	liveDocuments := objectField(t, live, "documents")
	assertFacetTombstone(t, arrayField(t, liveDocuments, "facets"), "", "region")
	assertFacetTombstone(t, arrayField(t, liveDocuments, "facet-documents"), "region", "default")
}

func arrayField(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	items, ok := value[key].([]any)
	if !ok {
		t.Fatalf("field %s is not an array: %#v", key, value[key])
	}
	return items
}

func assertFacetTombstone(t *testing.T, items []any, facetIdentity, identity string) {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok || stringField(t, item, "identity") != identity || (facetIdentity != "" && stringField(t, item, "facetIdentity") != facetIdentity) {
			continue
		}
		state := objectField(t, item, "state")
		if state["deletedAt"] == nil {
			t.Fatalf("%s is not a tombstone: %#v", identity, item)
		}
		return
	}
	t.Fatalf("tombstone %s not found", identity)
}

func assertPortableFacetTombstone(t *testing.T, items []any, facetIdentity, identity string) {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok || stringField(t, item, "identity") != identity || (facetIdentity != "" && stringField(t, item, "facetIdentity") != facetIdentity) {
			continue
		}
		if deleted, _ := item["deleted"].(bool); !deleted {
			t.Fatalf("%s is not a portable tombstone: %#v", identity, item)
		}
		return
	}
	t.Fatalf("portable tombstone %s not found", identity)
}
