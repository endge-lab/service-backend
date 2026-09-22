//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestRevisionDetailHTTPContract verifies that a stored document revision is
// readable only inside the caller's workspace and cannot leak across scopes.
func TestRevisionDetailHTTPContract(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	created := perform(t, fixture.app, http.MethodPost, "/api/v1/queries", map[string]any{
		"identity": "revision-detail", "displayName": "Revision detail", "source": "query {}", "sourceVersion": 2,
	}, fixture.workspaceAdmin)
	assertStatus(t, created, fiber.StatusCreated)
	created.Body.Close()
	patchHeaders := cloneHeaders(fixture.workspaceAdmin)
	patchHeaders[fiber.HeaderIfMatch] = `"1"`
	patched := perform(t, fixture.app, http.MethodPatch, "/api/v1/queries/revision-detail", map[string]any{"source": "query { updated }"}, patchHeaders)
	assertStatus(t, patched, fiber.StatusOK)
	patched.Body.Close()

	revisions := perform(t, fixture.app, http.MethodGet, "/api/v1/domain/documents/queries/revision-detail/revisions", nil, fixture.viewer)
	assertStatus(t, revisions, fiber.StatusOK)
	items := listItems(t, decodeObject(t, revisions))
	if len(items) != 2 {
		t.Fatalf("revision list count=%d, expected 2", len(items))
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("revision item is not an object: %#v", items[0])
	}
	revisionID := stringField(t, first, "id")
	path := "/api/v1/domain/documents/queries/revision-detail/revisions/" + revisionID

	unauthenticated := perform(t, fixture.app, http.MethodGet, path, nil, map[string]string{"X-Endge-Workspace": fixture.workspace})
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()
	viewer := perform(t, fixture.app, http.MethodGet, path, nil, fixture.viewer)
	assertStatus(t, viewer, fiber.StatusOK)
	if stringField(t, decodeObject(t, viewer), "id") != revisionID {
		t.Fatal("revision detail returned another revision")
	}
	outsider := perform(t, fixture.app, http.MethodGet, path, nil, fixture.outsider)
	assertStatus(t, outsider, fiber.StatusForbidden)
	outsider.Body.Close()
	missing := perform(t, fixture.app, http.MethodGet, "/api/v1/domain/documents/queries/revision-detail/revisions/00000000-0000-0000-0000-000000000099", nil, fixture.viewer)
	assertStatus(t, missing, fiber.StatusNotFound)
	missing.Body.Close()
	isolationHeaders := cloneHeaders(fixture.platform)
	isolationHeaders["X-Endge-Workspace"] = fixture.isolatedWorkspace
	isolation := perform(t, fixture.app, http.MethodGet, path, nil, isolationHeaders)
	assertStatus(t, isolation, fiber.StatusNotFound)
	isolation.Body.Close()
}
