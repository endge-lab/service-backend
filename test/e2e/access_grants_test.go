//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestWorkspaceAccessGrantLifecycle verifies that a workspace administrator can
// grant and revoke a role, and that the change immediately affects HTTP access.
func TestWorkspaceAccessGrantLifecycle(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	baseURL := "/api/v1/access-grants"
	workspaceQuery := "?scopeType=workspace&workspaceIdentity=" + fixture.workspace

	unauthenticatedList := perform(t, fixture.app, http.MethodGet, baseURL+workspaceQuery, nil, nil)
	assertStatus(t, unauthenticatedList, fiber.StatusUnauthorized)
	unauthenticatedList.Body.Close()
	invalidScope := perform(t, fixture.app, http.MethodGet, baseURL+"?scopeType=unknown", nil, fixture.platform)
	assertStatus(t, invalidScope, fiber.StatusBadRequest)
	invalidScope.Body.Close()
	viewerList := perform(t, fixture.app, http.MethodGet, baseURL+workspaceQuery, nil, fixture.viewer)
	assertStatus(t, viewerList, fiber.StatusForbidden)
	viewerList.Body.Close()

	beforeGrant := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, fixture.outsider)
	assertStatus(t, beforeGrant, fiber.StatusForbidden)
	beforeGrant.Body.Close()

	payload := map[string]any{
		"userId": fixture.outsiderID, "scopeType": "workspace",
		"workspaceIdentity": fixture.workspace, "role": "viewer",
	}
	unauthenticatedPut := perform(t, fixture.app, http.MethodPut, baseURL, payload, nil)
	assertStatus(t, unauthenticatedPut, fiber.StatusUnauthorized)
	unauthenticatedPut.Body.Close()
	viewerPut := perform(t, fixture.app, http.MethodPut, baseURL, payload, fixture.viewer)
	assertStatus(t, viewerPut, fiber.StatusForbidden)
	viewerPut.Body.Close()

	created := perform(t, fixture.app, http.MethodPut, baseURL, payload, fixture.workspaceAdmin)
	assertStatus(t, created, fiber.StatusOK)
	grantID := stringField(t, decodeObject(t, created), "id")

	listed := perform(t, fixture.app, http.MethodGet, baseURL+workspaceQuery, nil, fixture.workspaceAdmin)
	assertStatus(t, listed, fiber.StatusOK)
	if !hasID(t, listItems(t, decodeObject(t, listed)), grantID) {
		t.Fatal("created workspace grant is absent from its scope list")
	}

	readAfterGrant := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, fixture.outsider)
	assertStatus(t, readAfterGrant, fiber.StatusOK)
	readAfterGrant.Body.Close()
	writeAfterGrant := perform(t, fixture.app, http.MethodPost, "/api/v1/queries", map[string]any{
		"identity": "grant-viewer-write", "displayName": "Denied viewer write", "source": "query {}", "sourceVersion": 2,
	}, fixture.outsider)
	assertStatus(t, writeAfterGrant, fiber.StatusForbidden)
	writeAfterGrant.Body.Close()
	isolationHeaders := cloneHeaders(fixture.outsider)
	isolationHeaders["X-Endge-Workspace"] = fixture.isolatedWorkspace
	isolation := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, isolationHeaders)
	assertStatus(t, isolation, fiber.StatusForbidden)
	isolation.Body.Close()

	unauthenticatedDelete := perform(t, fixture.app, http.MethodDelete, baseURL+"/"+grantID, nil, nil)
	assertStatus(t, unauthenticatedDelete, fiber.StatusUnauthorized)
	unauthenticatedDelete.Body.Close()
	deleted := perform(t, fixture.app, http.MethodDelete, baseURL+"/"+grantID, nil, fixture.workspaceAdmin)
	assertStatus(t, deleted, fiber.StatusNoContent)
	deleted.Body.Close()

	afterRevoke := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, fixture.outsider)
	assertStatus(t, afterRevoke, fiber.StatusForbidden)
	afterRevoke.Body.Close()
}

// TestPlatformAccessGrantLifecycle verifies that a platform grant immediately
// enables a platform-only operation and revoking it removes that permission.
func TestPlatformAccessGrantLifecycle(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	baseURL := "/api/v1/access-grants"
	backendURL := "/api/v1/backend-connections"

	beforeGrant := perform(t, fixture.app, http.MethodPost, backendURL, map[string]any{
		"name": "Denied platform backend", "baseUrl": "https://denied.example.com",
	}, fixture.outsider)
	assertStatus(t, beforeGrant, fiber.StatusForbidden)
	beforeGrant.Body.Close()

	created := perform(t, fixture.app, http.MethodPut, baseURL, map[string]any{
		"userId": fixture.outsiderID, "scopeType": "platform", "role": "admin",
	}, fixture.platform)
	assertStatus(t, created, fiber.StatusOK)
	grantID := stringField(t, decodeObject(t, created), "id")

	listed := perform(t, fixture.app, http.MethodGet, baseURL+"?scopeType=platform&userId="+fixture.outsiderID, nil, fixture.platform)
	assertStatus(t, listed, fiber.StatusOK)
	if !hasID(t, listItems(t, decodeObject(t, listed)), grantID) {
		t.Fatal("created platform grant is absent from platform grant list")
	}
	workspaceAdminList := perform(t, fixture.app, http.MethodGet, baseURL+"?scopeType=platform", nil, fixture.workspaceAdmin)
	assertStatus(t, workspaceAdminList, fiber.StatusForbidden)
	workspaceAdminList.Body.Close()

	afterGrant := perform(t, fixture.app, http.MethodPost, backendURL, map[string]any{
		"name": "Granted platform backend", "baseUrl": "https://granted.example.com",
	}, fixture.outsider)
	assertStatus(t, afterGrant, fiber.StatusCreated)
	afterGrant.Body.Close()

	deleted := perform(t, fixture.app, http.MethodDelete, baseURL+"/"+grantID, nil, fixture.platform)
	assertStatus(t, deleted, fiber.StatusNoContent)
	deleted.Body.Close()
	afterRevoke := perform(t, fixture.app, http.MethodPost, backendURL, map[string]any{
		"name": "Revoked platform backend", "baseUrl": "https://revoked.example.com",
	}, fixture.outsider)
	assertStatus(t, afterRevoke, fiber.StatusForbidden)
	afterRevoke.Body.Close()
}

// TestBulkWorkspaceAccessGrants verifies selected and all-active bulk upserts,
// their counters, role update semantics, validation and authorization.
func TestBulkWorkspaceAccessGrants(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	bulkURL := "/api/v1/access-grants/bulk-workspaces"
	selectedPayload := map[string]any{
		"userId": fixture.outsiderID, "role": "editor",
		"selection": map[string]any{"type": "selected", "workspaceIdentities": []string{fixture.workspace, fixture.isolatedWorkspace}},
	}

	unauthenticated := perform(t, fixture.app, http.MethodPost, bulkURL, selectedPayload, nil)
	assertStatus(t, unauthenticated, fiber.StatusUnauthorized)
	unauthenticated.Body.Close()
	viewer := perform(t, fixture.app, http.MethodPost, bulkURL, selectedPayload, fixture.viewer)
	assertStatus(t, viewer, fiber.StatusForbidden)
	viewer.Body.Close()
	invalidSelection := perform(t, fixture.app, http.MethodPost, bulkURL, map[string]any{
		"userId": fixture.outsiderID, "role": "viewer", "selection": map[string]any{"type": "selected"},
	}, fixture.platform)
	assertStatus(t, invalidSelection, fiber.StatusBadRequest)
	invalidSelection.Body.Close()

	selected := perform(t, fixture.app, http.MethodPost, bulkURL, selectedPayload, fixture.platform)
	assertStatus(t, selected, fiber.StatusOK)
	selectedResult := decodeObject(t, selected)
	if numberField(t, selectedResult, "affected") != 2 || numberField(t, selectedResult, "created") != 2 || numberField(t, selectedResult, "updated") != 0 {
		t.Fatalf("unexpected selected bulk result: %#v", selectedResult)
	}

	for _, workspace := range []string{fixture.workspace, fixture.isolatedWorkspace} {
		t.Run("selected editor can write "+workspace, func(t *testing.T) {
			headers := cloneHeaders(fixture.outsider)
			headers["X-Endge-Workspace"] = workspace
			response := perform(t, fixture.app, http.MethodPost, "/api/v1/queries", map[string]any{
				"identity": "bulk-editor-" + workspace, "displayName": "Bulk editor", "source": "query {}", "sourceVersion": 2,
			}, headers)
			assertStatus(t, response, fiber.StatusCreated)
			response.Body.Close()
		})
	}
	defaultHeaders := cloneHeaders(fixture.outsider)
	defaultHeaders["X-Endge-Workspace"] = "default"
	defaultDenied := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, defaultHeaders)
	assertStatus(t, defaultDenied, fiber.StatusForbidden)
	defaultDenied.Body.Close()

	selectedPayload["role"] = "viewer"
	updated := perform(t, fixture.app, http.MethodPost, bulkURL, selectedPayload, fixture.platform)
	assertStatus(t, updated, fiber.StatusOK)
	updatedResult := decodeObject(t, updated)
	if numberField(t, updatedResult, "affected") != 2 || numberField(t, updatedResult, "created") != 0 || numberField(t, updatedResult, "updated") != 2 {
		t.Fatalf("unexpected selected bulk update result: %#v", updatedResult)
	}
	viewerWrite := perform(t, fixture.app, http.MethodPost, "/api/v1/queries", map[string]any{
		"identity": "bulk-viewer-write", "displayName": "Denied bulk viewer write", "source": "query {}", "sourceVersion": 2,
	}, fixture.outsider)
	assertStatus(t, viewerWrite, fiber.StatusForbidden)
	viewerWrite.Body.Close()

	workspaces := perform(t, fixture.app, http.MethodGet, "/api/v1/workspaces", nil, fixture.platform)
	assertStatus(t, workspaces, fiber.StatusOK)
	workspaceCount := len(listItems(t, decodeObject(t, workspaces)))
	allActive := perform(t, fixture.app, http.MethodPost, bulkURL, map[string]any{
		"userId": fixture.outsiderID, "role": "viewer", "selection": map[string]any{"type": "all-active"},
	}, fixture.platform)
	assertStatus(t, allActive, fiber.StatusOK)
	allActiveResult := decodeObject(t, allActive)
	if numberField(t, allActiveResult, "affected") != workspaceCount || numberField(t, allActiveResult, "created") != workspaceCount-2 || numberField(t, allActiveResult, "updated") != 2 {
		t.Fatalf("unexpected all-active bulk result: workspaces=%d result=%#v", workspaceCount, allActiveResult)
	}
	defaultAllowed := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, defaultHeaders)
	assertStatus(t, defaultAllowed, fiber.StatusOK)
	defaultAllowed.Body.Close()
}
