//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// TestWorkspaceRBAC проверяет роли, platform admin и отсутствие неявных прав на default.
func TestWorkspaceRBAC(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))
	adminHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "platform-admin", Username: "admin", DisplayName: "Admin", Groups: []string{"endge-platform-admins"}}))
	viewerHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "viewer", Username: "viewer", DisplayName: "Viewer"}))
	editorHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "editor", Username: "editor", DisplayName: "Editor"}))

	_ = currentUserID(t, app, adminHeaders)
	viewerID := currentUserID(t, app, viewerHeaders)
	editorID := currentUserID(t, app, editorHeaders)

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
	withoutDefaultMembership := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "without-default-membership", "displayName": "Without membership", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, withoutDefaultMembership, fiber.StatusForbidden)
	putMembership(t, app, adminHeaders, "default", viewerID, "editor")
	explicitEditor := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "explicit-editor", "displayName": "Explicit editor", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, explicitEditor, fiber.StatusCreated)
	putMembership(t, app, adminHeaders, "default", viewerID, "viewer")
	explicitViewer := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "explicit-viewer", "displayName": "Explicit", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, explicitViewer, fiber.StatusForbidden)
	deleteMembership := perform(t, app, http.MethodDelete, "/api/v1/workspaces/default/members/"+viewerID, nil, adminHeaders)
	assertStatus(t, deleteMembership, fiber.StatusNoContent)
	withoutMembershipAgain := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "without-membership-again", "displayName": "Without membership again", "source": "query {}", "sourceVersion": 2}, defaultHeaders)
	assertStatus(t, withoutMembershipAgain, fiber.StatusForbidden)

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

// TestSensitiveWorkspaceMutationsRequireAdmin проверяет полную матрицу ролей
// для commits и AuthProfile credential references через настоящий HTTP pipeline.
func TestSensitiveWorkspaceMutationsRequireAdmin(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))

	platform := bearer(provider.Token(t, support.TokenInput{Subject: "security-platform", Username: "security-platform", DisplayName: "Security Platform", Groups: []string{"endge-platform-admins"}}))
	admin := bearer(provider.Token(t, support.TokenInput{Subject: "security-admin", Username: "security-admin", DisplayName: "Security Admin"}))
	editor := bearer(provider.Token(t, support.TokenInput{Subject: "security-editor", Username: "security-editor", DisplayName: "Security Editor"}))
	viewer := bearer(provider.Token(t, support.TokenInput{Subject: "security-viewer", Username: "security-viewer", DisplayName: "Security Viewer"}))
	outsider := bearer(provider.Token(t, support.TokenInput{Subject: "security-outsider", Username: "security-outsider", DisplayName: "Security Outsider"}))

	_ = currentUserID(t, app, platform)
	adminID := currentUserID(t, app, admin)
	editorID := currentUserID(t, app, editor)
	viewerID := currentUserID(t, app, viewer)
	_ = currentUserID(t, app, outsider)

	workspace := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{"identity": "security-rbac", "displayName": "Security RBAC", "dataMode": "development"}, platform)
	assertStatus(t, workspace, fiber.StatusCreated)
	workspace.Body.Close()
	putMembership(t, app, platform, "security-rbac", adminID, "admin")
	putMembership(t, app, platform, "security-rbac", editorID, "editor")
	putMembership(t, app, platform, "security-rbac", viewerID, "viewer")

	adminWorkspace := workspaceHeaders(admin, "security-rbac")
	platformWorkspace := workspaceHeaders(platform, "security-rbac")
	editorWorkspace := workspaceHeaders(editor, "security-rbac")
	viewerWorkspace := workspaceHeaders(viewer, "security-rbac")
	outsiderWorkspace := workspaceHeaders(outsider, "security-rbac")
	unauthenticatedWorkspace := map[string]string{"X-Endge-Workspace": "security-rbac"}

	denied := []struct {
		name    string
		slug    string
		headers map[string]string
		status  int
	}{
		{name: "unauthenticated", slug: "unauthenticated", headers: unauthenticatedWorkspace, status: fiber.StatusUnauthorized},
		{name: "without membership", slug: "without-membership", headers: outsiderWorkspace, status: fiber.StatusForbidden},
		{name: "viewer", slug: "viewer", headers: viewerWorkspace, status: fiber.StatusForbidden},
		{name: "editor", slug: "editor", headers: editorWorkspace, status: fiber.StatusForbidden},
	}

	authProfilePayload := func(identity, passwordRef string) map[string]any {
		return map[string]any{
			"identity": identity, "displayName": identity, "adapterId": "oidc", "config": map[string]any{},
			"credentialRefs": map[string]any{"password": passwordRef}, "persist": "memory",
		}
	}

	t.Run("AuthProfile reads follow workspace read access", func(t *testing.T) {
		created := perform(t, app, http.MethodPost, "/api/v1/auth-profiles", authProfilePayload("security-readable", "PASSWORD_READABLE"), adminWorkspace)
		assertStatus(t, created, fiber.StatusCreated)
		created.Body.Close()

		for _, role := range []struct {
			name    string
			headers map[string]string
			status  int
		}{
			{name: "unauthenticated", headers: unauthenticatedWorkspace, status: fiber.StatusUnauthorized},
			{name: "without membership", headers: outsiderWorkspace, status: fiber.StatusForbidden},
			{name: "viewer", headers: viewerWorkspace, status: fiber.StatusOK},
			{name: "editor", headers: editorWorkspace, status: fiber.StatusOK},
			{name: "admin", headers: adminWorkspace, status: fiber.StatusOK},
			{name: "platform admin", headers: platformWorkspace, status: fiber.StatusOK},
		} {
			t.Run(role.name, func(t *testing.T) {
				response := perform(t, app, http.MethodGet, "/api/v1/auth-profiles/security-readable", nil, role.headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
		}
	})

	t.Run("AuthProfile mutations deny every role below admin", func(t *testing.T) {
		for _, role := range denied {
			t.Run("create "+role.name, func(t *testing.T) {
				response := perform(t, app, http.MethodPost, "/api/v1/auth-profiles", authProfilePayload("denied-create-"+role.slug, "PASSWORD_DENIED"), role.headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
		}

		created := perform(t, app, http.MethodPost, "/api/v1/auth-profiles", authProfilePayload("security-protected", "PASSWORD_V1"), adminWorkspace)
		assertStatus(t, created, fiber.StatusCreated)
		etag := created.Header.Get("ETag")
		created.Body.Close()

		for _, role := range denied {
			t.Run("patch "+role.name, func(t *testing.T) {
				headers := cloneHeaders(role.headers)
				headers["If-Match"] = etag
				response := perform(t, app, http.MethodPatch, "/api/v1/auth-profiles/security-protected", map[string]any{"credentialRefs": map[string]any{"password": "PASSWORD_V2"}}, headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
			t.Run("delete "+role.name, func(t *testing.T) {
				headers := cloneHeaders(role.headers)
				headers["If-Match"] = etag
				response := perform(t, app, http.MethodDelete, "/api/v1/auth-profiles/security-protected", nil, headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
			t.Run("restore "+role.name, func(t *testing.T) {
				headers := cloneHeaders(role.headers)
				headers["If-Match"] = etag
				response := perform(t, app, http.MethodPost, "/api/v1/auth-profiles/security-protected/restore", nil, headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
		}
	})

	for _, privileged := range []struct {
		name    string
		headers map[string]string
	}{
		{name: "admin", headers: adminWorkspace},
		{name: "platform-admin", headers: platformWorkspace},
	} {
		t.Run("AuthProfile lifecycle "+privileged.name, func(t *testing.T) {
			identity := "security-lifecycle-" + privileged.name
			created := perform(t, app, http.MethodPost, "/api/v1/auth-profiles", authProfilePayload(identity, "PASSWORD_V1"), privileged.headers)
			assertStatus(t, created, fiber.StatusCreated)
			etag := created.Header.Get("ETag")
			created.Body.Close()

			patchHeaders := cloneHeaders(privileged.headers)
			patchHeaders["If-Match"] = etag
			patched := perform(t, app, http.MethodPatch, "/api/v1/auth-profiles/"+identity, map[string]any{"credentialRefs": map[string]any{"password": "PASSWORD_V2"}}, patchHeaders)
			assertStatus(t, patched, fiber.StatusOK)
			etag = patched.Header.Get("ETag")
			patched.Body.Close()

			deleteHeaders := cloneHeaders(privileged.headers)
			deleteHeaders["If-Match"] = etag
			deleted := perform(t, app, http.MethodDelete, "/api/v1/auth-profiles/"+identity, nil, deleteHeaders)
			assertStatus(t, deleted, fiber.StatusOK)
			etag = deleted.Header.Get("ETag")
			deleted.Body.Close()

			restoreHeaders := cloneHeaders(privileged.headers)
			restoreHeaders["If-Match"] = etag
			restored := perform(t, app, http.MethodPost, "/api/v1/auth-profiles/"+identity+"/restore", nil, restoreHeaders)
			assertStatus(t, restored, fiber.StatusOK)
			restored.Body.Close()
		})

		t.Run("raw password rejected for "+privileged.name, func(t *testing.T) {
			payload := authProfilePayload("raw-password-"+privileged.name, "PASSWORD_REF")
			payload["config"] = map[string]any{"password": "raw-password"}
			response := perform(t, app, http.MethodPost, "/api/v1/auth-profiles", payload, privileged.headers)
			assertStatus(t, response, fiber.StatusBadRequest)
			if code := stringField(t, decodeObject(t, response), "code"); code != "secret_field_forbidden" {
				t.Fatalf("raw password error code=%q", code)
			}
		})
	}

	t.Run("commit plan and history remain readable", func(t *testing.T) {
		pending := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "commit-state-one", "displayName": "Commit state one", "source": "query { one }", "sourceVersion": 2}, editorWorkspace)
		assertStatus(t, pending, fiber.StatusCreated)
		pending.Body.Close()

		for _, role := range []struct {
			name    string
			headers map[string]string
			status  int
		}{
			{name: "unauthenticated", headers: unauthenticatedWorkspace, status: fiber.StatusUnauthorized},
			{name: "without membership", headers: outsiderWorkspace, status: fiber.StatusForbidden},
			{name: "viewer", headers: viewerWorkspace, status: fiber.StatusOK},
			{name: "editor", headers: editorWorkspace, status: fiber.StatusOK},
			{name: "admin", headers: adminWorkspace, status: fiber.StatusOK},
			{name: "platform admin", headers: platformWorkspace, status: fiber.StatusOK},
		} {
			t.Run(role.name, func(t *testing.T) {
				response := perform(t, app, http.MethodPost, "/api/v1/commits/plan", nil, role.headers)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
		}
	})

	commitRequest := func(headers map[string]string, message string) *http.Response {
		return perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{
			"message": message, "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, adminWorkspace),
		}, headers)
	}

	t.Run("commit creation denies every role below admin", func(t *testing.T) {
		for _, role := range denied {
			t.Run(role.name, func(t *testing.T) {
				response := commitRequest(role.headers, "Denied "+role.name)
				assertStatus(t, response, role.status)
				response.Body.Close()
			})
		}
	})

	adminCommitResponse := commitRequest(adminWorkspace, "Admin commit")
	assertStatus(t, adminCommitResponse, fiber.StatusCreated)
	adminCommitID := stringField(t, decodeObject(t, adminCommitResponse), "id")

	t.Run("commit history read matrix", func(t *testing.T) {
		for _, role := range []struct {
			name    string
			headers map[string]string
			status  int
		}{
			{name: "unauthenticated", headers: unauthenticatedWorkspace, status: fiber.StatusUnauthorized},
			{name: "without membership", headers: outsiderWorkspace, status: fiber.StatusForbidden},
			{name: "viewer", headers: viewerWorkspace, status: fiber.StatusOK},
			{name: "editor", headers: editorWorkspace, status: fiber.StatusOK},
			{name: "admin", headers: adminWorkspace, status: fiber.StatusOK},
			{name: "platform admin", headers: platformWorkspace, status: fiber.StatusOK},
		} {
			t.Run(role.name, func(t *testing.T) {
				for _, target := range []string{"/api/v1/commits", "/api/v1/commits/" + adminCommitID, "/api/v1/commits/" + adminCommitID + "/diff"} {
					response := perform(t, app, http.MethodGet, target, nil, role.headers)
					assertStatus(t, response, role.status)
					response.Body.Close()
				}
			})
		}
	})

	secondPending := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "commit-state-two", "displayName": "Commit state two", "source": "query { two }", "sourceVersion": 2}, editorWorkspace)
	assertStatus(t, secondPending, fiber.StatusCreated)
	secondPending.Body.Close()
	platformCommitResponse := commitRequest(platformWorkspace, "Platform commit")
	assertStatus(t, platformCommitResponse, fiber.StatusCreated)
	platformCommitID := stringField(t, decodeObject(t, platformCommitResponse), "id")

	t.Run("commit restore requires admin", func(t *testing.T) {
		for _, role := range denied {
			t.Run(role.name, func(t *testing.T) {
				plan := perform(t, app, http.MethodPost, "/api/v1/commits/"+adminCommitID+"/restore/plan", nil, role.headers)
				assertStatus(t, plan, role.status)
				plan.Body.Close()
				restore := perform(t, app, http.MethodPost, "/api/v1/commits/"+adminCommitID+"/restore", map[string]any{"expectedHeadSequence": currentHeadSequence(t, app, adminWorkspace)}, role.headers)
				assertStatus(t, restore, role.status)
				restore.Body.Close()
			})
		}

		adminPlan := perform(t, app, http.MethodPost, "/api/v1/commits/"+adminCommitID+"/restore/plan", nil, adminWorkspace)
		assertStatus(t, adminPlan, fiber.StatusOK)
		adminPlan.Body.Close()
		adminRestore := perform(t, app, http.MethodPost, "/api/v1/commits/"+adminCommitID+"/restore", map[string]any{"expectedHeadSequence": currentHeadSequence(t, app, adminWorkspace)}, adminWorkspace)
		assertStatus(t, adminRestore, fiber.StatusCreated)
		adminRestore.Body.Close()

		platformPlan := perform(t, app, http.MethodPost, "/api/v1/commits/"+platformCommitID+"/restore/plan", nil, platformWorkspace)
		assertStatus(t, platformPlan, fiber.StatusOK)
		platformPlan.Body.Close()
		platformRestore := perform(t, app, http.MethodPost, "/api/v1/commits/"+platformCommitID+"/restore", map[string]any{"expectedHeadSequence": currentHeadSequence(t, app, platformWorkspace)}, platformWorkspace)
		assertStatus(t, platformRestore, fiber.StatusCreated)
		platformRestore.Body.Close()
	})

	t.Run("default without role and viewer cannot commit", func(t *testing.T) {
		withoutRole := bearer(provider.Token(t, support.TokenInput{Subject: "security-default-without-role", Username: "security-default-without-role", DisplayName: "Security Default Without Role"}))
		withoutRoleID := currentUserID(t, app, withoutRole)
		withoutRoleDefault := workspaceHeaders(withoutRole, "default")
		platformDefault := workspaceHeaders(platform, "default")
		response := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "Without role commit", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, platformDefault)}, withoutRoleDefault)
		assertStatus(t, response, fiber.StatusForbidden)
		response.Body.Close()

		putMembership(t, app, platform, "default", withoutRoleID, "viewer")
		explicitViewer := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "Explicit viewer commit", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, app, platformDefault)}, withoutRoleDefault)
		assertStatus(t, explicitViewer, fiber.StatusForbidden)
		explicitViewer.Body.Close()
	})
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

func workspaceHeaders(headers map[string]string, identity string) map[string]string {
	result := cloneHeaders(headers)
	result["X-Endge-Workspace"] = identity
	return result
}
