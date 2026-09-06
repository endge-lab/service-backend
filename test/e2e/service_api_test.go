//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// TestVersionHTTPContract verifies that the public version endpoint does not
// require an authenticated session and returns the service build metadata.
func TestVersionHTTPContract(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())

	response := perform(t, app, http.MethodGet, "/version", nil, nil)
	assertStatus(t, response, fiber.StatusOK)
	body := decodeObject(t, response)
	if stringField(t, body, "service") == "" || stringField(t, body, "version") == "" || stringField(t, body, "env") == "" {
		t.Fatalf("version response does not contain build metadata: %#v", body)
	}
	if _, ok := body["workspaceSchemaVersion"].(float64); !ok {
		t.Fatalf("version response does not contain workspaceSchemaVersion: %#v", body)
	}
	if _, ok := body["services"].([]any); !ok {
		t.Fatalf("version response does not contain services list: %#v", body)
	}
}

// TestServiceUsersSearchHTTPContract verifies search input validation and that
// only a platform or workspace administrator may search assignable users.
func TestServiceUsersSearchHTTPContract(t *testing.T) {
	fixture := newServiceAPIFixture(t)

	unauthenticated := perform(t, fixture.app, http.MethodGet, "/api/v1/service-users/search?q=service", nil, nil)
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()

	invalid := perform(t, fixture.app, http.MethodGet, "/api/v1/service-users/search?q=x", nil, fixture.platform)
	assertStatus(t, invalid, fiber.StatusBadRequest)
	invalid.Body.Close()

	for _, actor := range []struct {
		name    string
		headers map[string]string
	}{
		{name: "viewer", headers: fixture.viewer},
		{name: "outsider", headers: fixture.outsider},
	} {
		t.Run(actor.name+" is forbidden", func(t *testing.T) {
			response := perform(t, fixture.app, http.MethodGet, "/api/v1/service-users/search?q=service&workspaceIdentity="+fixture.workspace, nil, actor.headers)
			assertStatus(t, response, fiber.StatusForbidden)
			response.Body.Close()
		})
	}

	platform := perform(t, fixture.app, http.MethodGet, "/api/v1/service-users/search?q=service&limit=1", nil, fixture.platform)
	assertStatus(t, platform, fiber.StatusOK)
	if len(listItems(t, decodeObject(t, platform))) != 1 {
		t.Fatal("service-user search did not apply the requested page size")
	}

	workspaceAdmin := perform(t, fixture.app, http.MethodGet, "/api/v1/service-users/search?q=service&workspaceIdentity="+fixture.workspace, nil, fixture.workspaceAdmin)
	assertStatus(t, workspaceAdmin, fiber.StatusOK)
	if !hasUsername(t, listItems(t, decodeObject(t, workspaceAdmin)), "service-api-viewer") {
		t.Fatal("workspace admin cannot see an existing assignable user")
	}
}

// TestWorkspaceHTTPContracts verifies workspace list/get/patch/member routes,
// role boundaries and that a user cannot discover an unrelated workspace.
func TestWorkspaceHTTPContracts(t *testing.T) {
	fixture := newServiceAPIFixture(t)

	unauthenticated := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces", nil, nil)
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()

	viewerList := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces", nil, fixture.viewer)
	assertStatus(t, viewerList, fiber.StatusOK)
	viewerItems := listItems(t, decodeObject(t, viewerList))
	if !hasIdentity(t, viewerItems, fixture.workspace) || hasIdentity(t, viewerItems, fixture.isolatedWorkspace) {
		t.Fatalf("workspace list violates isolation: %#v", viewerItems)
	}

	for _, actor := range []struct {
		name    string
		headers map[string]string
	}{
		{name: "workspace admin", headers: fixture.workspaceAdmin},
		{name: "viewer", headers: fixture.viewer},
	} {
		t.Run("get is allowed for "+actor.name, func(t *testing.T) {
			response := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces/"+fixture.workspace, nil, actor.headers)
			assertStatus(t, response, fiber.StatusOK)
			if stringField(t, decodeObject(t, response), "identity") != fixture.workspace {
				t.Fatal("workspace response has an unexpected identity")
			}
		})
	}

	forbiddenGet := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces/"+fixture.isolatedWorkspace, nil, fixture.outsider)
	assertStatus(t, forbiddenGet, fiber.StatusForbidden)
	forbiddenGet.Body.Close()

	get := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces/"+fixture.workspace, nil, fixture.workspaceAdmin)
	assertStatus(t, get, fiber.StatusOK)
	etag := get.Header.Get(fiber.HeaderETag)
	get.Body.Close()

	missingPrecondition := perform(t, fixture.app, http.MethodPatch, "/api/v1/workspaces/"+fixture.workspace, map[string]any{"displayName": "Changed workspace"}, fixture.workspaceAdmin)
	assertStatus(t, missingPrecondition, fiber.StatusPreconditionRequired)
	missingPrecondition.Body.Close()

	viewerPatchHeaders := cloneHeaders(fixture.viewer)
	viewerPatchHeaders[fiber.HeaderIfMatch] = etag
	viewerPatch := perform(t, fixture.app, http.MethodPatch, "/api/v1/workspaces/"+fixture.workspace, map[string]any{"displayName": "Forbidden workspace change"}, viewerPatchHeaders)
	assertStatus(t, viewerPatch, fiber.StatusForbidden)
	viewerPatch.Body.Close()

	patchHeaders := cloneHeaders(fixture.workspaceAdmin)
	patchHeaders[fiber.HeaderIfMatch] = etag
	patched := perform(t, fixture.app, http.MethodPatch, "/api/v1/workspaces/"+fixture.workspace, map[string]any{"displayName": "Changed workspace"}, patchHeaders)
	assertStatus(t, patched, fiber.StatusOK)
	newETag := patched.Header.Get(fiber.HeaderETag)
	if stringField(t, decodeObject(t, patched), "displayName") != "Changed workspace" || newETag == etag {
		t.Fatalf("workspace patch did not persist a new revision: old=%q new=%q", etag, newETag)
	}

	stale := perform(t, fixture.app, http.MethodPatch, "/api/v1/workspaces/"+fixture.workspace, map[string]any{"displayName": "Stale workspace change"}, patchHeaders)
	assertStatus(t, stale, fiber.StatusConflict)
	stale.Body.Close()

	members := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces/"+fixture.workspace+"/members", nil, fixture.workspaceAdmin)
	assertStatus(t, members, fiber.StatusOK)
	if len(listItems(t, decodeObject(t, members))) != 2 {
		t.Fatal("workspace member list must contain the two explicit memberships")
	}
	viewerMembers := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces/"+fixture.workspace+"/members", nil, fixture.viewer)
	assertStatus(t, viewerMembers, fiber.StatusForbidden)
	viewerMembers.Body.Close()
}

// TestIntegrationHTTPContracts verifies the global integration catalog
// lifecycle, ETag concurrency, soft deletion and platform-admin mutations.
func TestIntegrationHTTPContracts(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	identity := "service-api-integration"
	baseURL := "/api/v1/integrations/" + identity

	unauthenticated := perform(t, fixture.app, http.MethodGet, "/api/v1/integrations", nil, nil)
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()

	createUnauthorized := perform(t, fixture.app, http.MethodPost, "/api/v1/integrations", map[string]any{"identity": identity, "displayName": "Service API integration", "version": "1.0.0"}, nil)
	assertStatus(t, createUnauthorized, fiber.StatusUnauthorized)
	createUnauthorized.Body.Close()
	createForbidden := perform(t, fixture.app, http.MethodPost, "/api/v1/integrations", map[string]any{"identity": identity, "displayName": "Service API integration", "version": "1.0.0"}, fixture.workspaceAdmin)
	assertStatus(t, createForbidden, fiber.StatusForbidden)
	createForbidden.Body.Close()

	created := perform(t, fixture.app, http.MethodPost, "/api/v1/integrations", map[string]any{"identity": identity, "displayName": "Service API integration", "version": "1.0.0"}, fixture.platform)
	assertStatus(t, created, fiber.StatusCreated)
	etag := created.Header.Get(fiber.HeaderETag)
	if stringField(t, decodeObject(t, created), "identity") != identity || etag == "" {
		t.Fatal("integration create response does not contain identity and ETag")
	}

	viewerList := perform(t, fixture.app, http.MethodGet, "/api/v1/integrations", nil, fixture.viewer)
	assertStatus(t, viewerList, fiber.StatusOK)
	if !hasIdentity(t, listItems(t, decodeObject(t, viewerList)), identity) {
		t.Fatal("authenticated viewer cannot read global integration catalog")
	}
	viewerGet := perform(t, fixture.app, http.MethodGet, baseURL, nil, fixture.viewer)
	assertStatus(t, viewerGet, fiber.StatusOK)
	viewerGet.Body.Close()

	missingPrecondition := perform(t, fixture.app, http.MethodPatch, baseURL, map[string]any{"displayName": "Changed integration"}, fixture.platform)
	assertStatus(t, missingPrecondition, fiber.StatusPreconditionRequired)
	missingPrecondition.Body.Close()
	forbiddenHeaders := cloneHeaders(fixture.workspaceAdmin)
	forbiddenHeaders[fiber.HeaderIfMatch] = etag
	forbiddenPatch := perform(t, fixture.app, http.MethodPatch, baseURL, map[string]any{"displayName": "Forbidden integration change"}, forbiddenHeaders)
	assertStatus(t, forbiddenPatch, fiber.StatusForbidden)
	forbiddenPatch.Body.Close()

	patchHeaders := cloneHeaders(fixture.platform)
	patchHeaders[fiber.HeaderIfMatch] = etag
	patched := perform(t, fixture.app, http.MethodPatch, baseURL, map[string]any{"displayName": "Changed integration"}, patchHeaders)
	assertStatus(t, patched, fiber.StatusOK)
	etag = patched.Header.Get(fiber.HeaderETag)
	if stringField(t, decodeObject(t, patched), "displayName") != "Changed integration" {
		t.Fatal("integration patch did not persist displayName")
	}

	stale := perform(t, fixture.app, http.MethodPatch, baseURL, map[string]any{"displayName": "Stale integration change"}, patchHeaders)
	assertStatus(t, stale, fiber.StatusConflict)
	stale.Body.Close()

	deleteHeaders := cloneHeaders(fixture.platform)
	deleteHeaders[fiber.HeaderIfMatch] = etag
	deleted := perform(t, fixture.app, http.MethodDelete, baseURL, nil, deleteHeaders)
	assertStatus(t, deleted, fiber.StatusOK)
	etag = deleted.Header.Get(fiber.HeaderETag)
	if decodeObject(t, deleted)["deletedAt"] == nil {
		t.Fatal("soft-deleted integration has no deletedAt")
	}

	hidden := perform(t, fixture.app, http.MethodGet, baseURL, nil, fixture.platform)
	assertStatus(t, hidden, fiber.StatusNotFound)
	hidden.Body.Close()
	included := perform(t, fixture.app, http.MethodGet, baseURL+"?includeDeleted=true", nil, fixture.platform)
	assertStatus(t, included, fiber.StatusOK)
	included.Body.Close()

	restoreHeaders := cloneHeaders(fixture.platform)
	restoreHeaders[fiber.HeaderIfMatch] = etag
	restored := perform(t, fixture.app, http.MethodPost, baseURL+"/restore", nil, restoreHeaders)
	assertStatus(t, restored, fiber.StatusOK)
	if decodeObject(t, restored)["deletedAt"] != nil {
		t.Fatal("restored integration still has deletedAt")
	}
}

// TestBackendConnectionsHTTPContracts verifies that the global catalog is
// readable to authenticated users but only platform administrators can mutate it.
func TestBackendConnectionsHTTPContracts(t *testing.T) {
	fixture := newServiceAPIFixture(t)

	unauthenticated := perform(t, fixture.app, http.MethodGet, "/api/v1/backend-connections", nil, nil)
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()

	viewerList := perform(t, fixture.app, http.MethodGet, "/api/v1/backend-connections", nil, fixture.viewer)
	assertStatus(t, viewerList, fiber.StatusOK)
	if decodeObject(t, viewerList)["canManage"] != false {
		t.Fatal("viewer unexpectedly received backend catalog management permission")
	}

	payload := map[string]any{"name": "  Service API backend  ", "baseUrl": "HTTPS://BACKEND.EXAMPLE.COM/api/"}
	createUnauthorized := perform(t, fixture.app, http.MethodPost, "/api/v1/backend-connections", payload, nil)
	assertStatus(t, createUnauthorized, fiber.StatusUnauthorized)
	createUnauthorized.Body.Close()
	createForbidden := perform(t, fixture.app, http.MethodPost, "/api/v1/backend-connections", payload, fixture.workspaceAdmin)
	assertStatus(t, createForbidden, fiber.StatusForbidden)
	createForbidden.Body.Close()

	created := perform(t, fixture.app, http.MethodPost, "/api/v1/backend-connections", payload, fixture.platform)
	assertStatus(t, created, fiber.StatusCreated)
	createdBody := decodeObject(t, created)
	id := stringField(t, createdBody, "id")
	if stringField(t, createdBody, "name") != "Service API backend" || stringField(t, createdBody, "baseUrl") != "https://backend.example.com/api" {
		t.Fatalf("backend connection was not normalized: %#v", createdBody)
	}

	platformList := perform(t, fixture.app, http.MethodGet, "/api/v1/backend-connections", nil, fixture.platform)
	assertStatus(t, platformList, fiber.StatusOK)
	platformListBody := decodeObject(t, platformList)
	if platformListBody["canManage"] != true || !hasID(t, listItems(t, platformListBody), id) {
		t.Fatalf("platform admin cannot see or manage created backend connection: %#v", platformListBody)
	}

	deleteForbidden := perform(t, fixture.app, http.MethodDelete, "/api/v1/backend-connections/"+id, nil, fixture.workspaceAdmin)
	assertStatus(t, deleteForbidden, fiber.StatusForbidden)
	deleteForbidden.Body.Close()
	deleted := perform(t, fixture.app, http.MethodDelete, "/api/v1/backend-connections/"+id, nil, fixture.platform)
	assertStatus(t, deleted, fiber.StatusNoContent)
	deleted.Body.Close()

	afterDelete := perform(t, fixture.app, http.MethodGet, "/api/v1/backend-connections", nil, fixture.platform)
	assertStatus(t, afterDelete, fiber.StatusOK)
	if hasID(t, listItems(t, decodeObject(t, afterDelete)), id) {
		t.Fatal("deleted backend connection remains in global catalog")
	}
}

func listItems(t *testing.T, value map[string]any) []any {
	t.Helper()
	items, ok := value["items"].([]any)
	if !ok {
		t.Fatalf("response does not contain items list: %#v", value)
	}
	return items
}

func hasIdentity(t *testing.T, items []any, identity string) bool {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("list item is not an object: %#v", raw)
		}
		if stringField(t, item, "identity") == identity {
			return true
		}
	}
	return false
}

func hasUsername(t *testing.T, items []any, username string) bool {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("list item is not an object: %#v", raw)
		}
		if stringField(t, item, "username") == username {
			return true
		}
	}
	return false
}

func hasID(t *testing.T, items []any, id string) bool {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("list item is not an object: %#v", raw)
		}
		if stringField(t, item, "id") == id {
			return true
		}
	}
	return false
}
