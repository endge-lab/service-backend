//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// TestWorkspaceRBACAndDefaultFallback проверяет роли, platform admin и особое правило default.
func TestWorkspaceRBACAndDefaultFallback(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))
	adminHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "platform-admin", Username: "admin", DisplayName: "Admin", Groups: []string{"endge-platform-admins"}}))
	viewerHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "viewer", Username: "viewer", DisplayName: "Viewer"}))
	editorHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "editor", Username: "editor", DisplayName: "Editor"}))

	viewerID := currentUserID(t, app, viewerHeaders)
	editorID := currentUserID(t, app, editorHeaders)
	_ = currentUserID(t, app, adminHeaders)

	workspace := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{"identity": "restricted", "displayName": "Restricted", "dataMode": "development"}, adminHeaders)
	assertStatus(t, workspace, fiber.StatusCreated)
	putMembership(t, app, adminHeaders, "restricted", viewerID, "viewer")
	putMembership(t, app, adminHeaders, "restricted", editorID, "editor")
	assertSessionWorkspaceRole(t, app, viewerHeaders, "restricted", "viewer")
	assertSessionWorkspaceRole(t, app, editorHeaders, "restricted", "editor")
	assertSessionWorkspaceRole(t, app, adminHeaders, "restricted", "admin")

	viewerRestricted := cloneHeaders(viewerHeaders)
	viewerRestricted["X-Endge-Workspace"] = "restricted"
	read := perform(t, app, http.MethodGet, "/api/v1/queries", nil, viewerRestricted)
	assertStatus(t, read, fiber.StatusOK)
	write := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "viewer-write", "displayName": "Viewer", "source": "query {}", "sourceVersion": 2}, viewerRestricted)
	assertStatus(t, write, fiber.StatusForbidden)

	editorRestricted := cloneHeaders(editorHeaders)
	editorRestricted["X-Endge-Workspace"] = "restricted"
	editorWrite := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "editor-write", "displayName": "Editor", "source": "query {}", "sourceVersion": 2}, editorRestricted)
	assertStatus(t, editorWrite, fiber.StatusCreated)

	noMembership := bearer(provider.Token(t, support.TokenInput{Subject: "outsider", Username: "outsider", DisplayName: "Outsider"}))
	noMembership["X-Endge-Workspace"] = "restricted"
	forbidden := perform(t, app, http.MethodGet, "/api/v1/queries", nil, noMembership)
	assertStatus(t, forbidden, fiber.StatusForbidden)

	defaultHeaders := cloneHeaders(viewerHeaders)
	defaultHeaders["X-Endge-Workspace"] = "default"
	implicitEditor := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "implicit-editor", "displayName": "Implicit", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, implicitEditor, fiber.StatusCreated)
	putMembership(t, app, adminHeaders, "default", viewerID, "viewer")
	explicitViewer := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "explicit-viewer", "displayName": "Explicit", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, explicitViewer, fiber.StatusForbidden)
	deleteMembership := perform(t, app, http.MethodDelete, "/api/v1/workspaces/default/members/"+viewerID, nil, adminHeaders)
	assertStatus(t, deleteMembership, fiber.StatusNoContent)
	implicitAgain := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "implicit-again", "displayName": "Implicit Again", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, implicitAgain, fiber.StatusCreated)

	integration := perform(t, app, http.MethodPost, "/api/v1/integrations", map[string]any{"identity": "test-integration", "displayName": "Test", "version": "1.0.0"}, adminHeaders)
	assertStatus(t, integration, fiber.StatusCreated)
	deniedIntegration := perform(t, app, http.MethodPost, "/api/v1/integrations", map[string]any{"identity": "forbidden-integration", "displayName": "Forbidden", "version": "1.0.0"}, editorHeaders)
	assertStatus(t, deniedIntegration, fiber.StatusForbidden)

	defaultStoreHeaders := cloneHeaders(adminHeaders)
	defaultStoreHeaders["X-Endge-Workspace"] = "default"
	defaultStore := perform(t, app, http.MethodPost, "/api/v1/stores", map[string]any{"identity": "default-only-store", "displayName": "Default Store", "source": "store {}", "sourceVersion": 1}, defaultStoreHeaders)
	assertStatus(t, defaultStore, fiber.StatusCreated)
	defaultStore.Body.Close()
	crossWorkspaceRelation := perform(t, app, http.MethodPost, "/api/v1/updates", map[string]any{"identity": "cross-workspace-update", "displayName": "Cross Workspace", "storeIdentity": "default-only-store", "source": "update {}", "sourceVersion": 1}, editorRestricted)
	assertStatus(t, crossWorkspaceRelation, fiber.StatusBadRequest)
}

func currentUserID(t *testing.T, app *fiber.App, headers map[string]string) string {
	t.Helper()
	response := perform(t, app, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, fiber.StatusOK)
	return stringField(t, objectField(t, decodeObject(t, response), "user"), "id")
}

func assertSessionWorkspaceRole(t *testing.T, app *fiber.App, headers map[string]string, identity, expectedRole string) {
	t.Helper()
	response := perform(t, app, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, fiber.StatusOK)
	workspaces, _ := decodeObject(t, response)["workspaces"].([]any)
	for _, raw := range workspaces {
		workspace, _ := raw.(map[string]any)
		if workspace["identity"] == identity {
			if role := stringField(t, workspace, "role"); role != expectedRole {
				t.Fatalf("workspace %s role=%q, ожидалась %q", identity, role, expectedRole)
			}
			return
		}
	}
	t.Fatalf("workspace %s отсутствует в session projection", identity)
}

func putMembership(t *testing.T, app *fiber.App, headers map[string]string, workspace, userID, role string) {
	t.Helper()
	response := perform(t, app, http.MethodPut, "/api/v1/workspaces/"+workspace+"/members/"+userID, map[string]any{"role": role}, headers)
	assertStatus(t, response, fiber.StatusOK)
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}
