//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// TestAccessTokensLifecycle проверяет выпуск PAT сессией, сужение прав токеном, пометку
// изменений агента, отзыв и истечение через полный HTTP pipeline.
func TestAccessTokensLifecycle(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))
	adminHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "platform-admin", Username: "admin", DisplayName: "Admin", Groups: []string{"endge-platform-admins"}}))
	editorHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "editor", Username: "editor", DisplayName: "Editor"}))
	viewerHeaders := bearer(provider.Token(t, support.TokenInput{Subject: "viewer", Username: "viewer", DisplayName: "Viewer"}))

	_ = currentUserID(t, app, adminHeaders)
	createWorkspace(t, app, adminHeaders, "afl")
	createWorkspace(t, app, adminHeaders, "kzn")
	editorID := currentUserID(t, app, editorHeaders)
	viewerID := currentUserID(t, app, viewerHeaders)
	putMembership(t, app, adminHeaders, "afl", editorID, "editor")
	putMembership(t, app, adminHeaders, "kzn", editorID, "editor")
	putMembership(t, app, adminHeaders, "afl", viewerID, "viewer")

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	// Ограничения выпуска.
	tooLong := perform(t, app, http.MethodPost, "/api/v1/access-tokens", map[string]any{"workspaceIdentity": "afl", "name": "long", "expiresAt": time.Now().Add(31 * 24 * time.Hour).UTC().Format(time.RFC3339)}, editorHeaders)
	assertErrorCode(t, tooLong, fiber.StatusBadRequest, "access_token_expiry_too_long")
	missingExpiry := perform(t, app, http.MethodPost, "/api/v1/access-tokens", map[string]any{"workspaceIdentity": "afl", "name": "no expiry"}, editorHeaders)
	assertStatus(t, missingExpiry, fiber.StatusBadRequest)
	missingExpiry.Body.Close()
	viewerWrite := perform(t, app, http.MethodPost, "/api/v1/access-tokens", map[string]any{"workspaceIdentity": "afl", "name": "w", "scope": "write", "expiresAt": expiresAt}, viewerHeaders)
	assertErrorCode(t, viewerWrite, fiber.StatusForbidden, "access_token_scope_exceeds_role")

	// Токен только для чтения: scope read по умолчанию.
	readSecret, readID := createAccessToken(t, app, editorHeaders, map[string]any{"workspaceIdentity": "afl", "name": "Agent read", "expiresAt": expiresAt}, "read")
	readHeaders := bearer(readSecret)
	assertStatus(t, perform(t, app, http.MethodGet, "/api/v1/queries", nil, workspaceHeaders(readHeaders, "afl")), fiber.StatusOK)
	assertSessionWorkspaceRole(t, app, readHeaders, "afl", "viewer")
	session := decodeObject(t, perform(t, app, http.MethodGet, "/api/session/me", nil, readHeaders))
	if workspaces, _ := session["workspaces"].([]any); len(workspaces) != 1 || session["platformAdmin"] != false {
		t.Fatalf("token session must expose only its workspace without platform admin: %v", session)
	}
	query := map[string]any{"identity": "agent-query", "displayName": "Agent", "source": "query {}", "sourceVersion": 2}
	assertErrorCode(t, perform(t, app, http.MethodPost, "/api/v1/queries", query, workspaceHeaders(readHeaders, "afl")), fiber.StatusForbidden, "access_token_scope_insufficient")
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/v1/queries", nil, workspaceHeaders(readHeaders, "kzn")), fiber.StatusForbidden, "access_token_workspace_forbidden")
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/v1/workspaces", nil, readHeaders), fiber.StatusForbidden, "access_token_route_forbidden")
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/v1/access-grants?scopeType=workspace&workspaceIdentity=afl", nil, readHeaders), fiber.StatusForbidden, "access_token_route_forbidden")
	assertErrorCode(t, perform(t, app, http.MethodPost, "/api/v1/access-tokens", map[string]any{"workspaceIdentity": "afl", "name": "child", "expiresAt": expiresAt}, readHeaders), fiber.StatusForbidden, "access_token_management_forbidden")

	// Токен записи без commit/delete: изменения помечаются, удаление и commit запрещены.
	writeSecret, writeID := createAccessToken(t, app, editorHeaders, map[string]any{"workspaceIdentity": "afl", "name": "Agent write", "scope": "write", "expiresAt": expiresAt}, "write")
	writeHeaders := workspaceHeaders(bearer(writeSecret), "afl")
	created := perform(t, app, http.MethodPost, "/api/v1/queries", query, writeHeaders)
	assertStatus(t, created, fiber.StatusCreated)
	created.Body.Close()
	revisions := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/documents/queries/agent-query/revisions", nil, workspaceHeaders(editorHeaders, "afl")))
	items, _ := revisions["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("revision of agent change is missing: %v", revisions)
	}
	latest, _ := items[0].(map[string]any)
	if latest["accessTokenId"] != writeID || stringField(t, objectField(t, latest, "createdBy"), "id") != editorID {
		t.Fatalf("agent revision must keep owner as author and mark token: %v", latest)
	}
	assertErrorCode(t, perform(t, app, http.MethodDelete, "/api/v1/queries/agent-query", nil, writeHeaders), fiber.StatusForbidden, "access_token_delete_forbidden")
	assertErrorCode(t, perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "agent"}, writeHeaders), fiber.StatusForbidden, "access_token_commit_forbidden")
	assertErrorCode(t, perform(t, app, http.MethodPut, "/api/v1/workspaces/afl/members/"+viewerID, map[string]any{"role": "editor"}, writeHeaders), fiber.StatusForbidden, "access_token_route_forbidden")

	// Список: секрет не возвращается; Platform Admin видит чужие токены.
	list := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/access-tokens", nil, editorHeaders))
	listed, _ := list["items"].([]any)
	if len(listed) != 2 || numberField(t, list, "maxTtlSeconds") != int((30*24*time.Hour)/time.Second) {
		t.Fatalf("unexpected token list: %v", list)
	}
	for _, raw := range listed {
		item, _ := raw.(map[string]any)
		if _, leaked := item["token"]; leaked || !strings.HasPrefix(stringField(t, item, "tokenPrefix"), "endge_pat_") {
			t.Fatalf("list must expose only display prefix: %v", item)
		}
		if item["id"] == readID && item["lastUsedAt"] == nil {
			t.Fatalf("lastUsedAt must be recorded after use: %v", item)
		}
	}
	assertStatus(t, perform(t, app, http.MethodGet, "/api/v1/access-tokens?owner=all", nil, editorHeaders), fiber.StatusForbidden)
	all := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/access-tokens?owner=all", nil, adminHeaders))
	if allItems, _ := all["items"].([]any); len(allItems) != 2 {
		t.Fatalf("platform admin must see all tokens: %v", all)
	}

	// Отзыв: чужой пользователь не видит токен, владелец отзывает, токен получает 401.
	assertStatus(t, perform(t, app, http.MethodDelete, "/api/v1/access-tokens/"+readID, nil, viewerHeaders), fiber.StatusNotFound)
	assertStatus(t, perform(t, app, http.MethodDelete, "/api/v1/access-tokens/"+readID, nil, editorHeaders), fiber.StatusNoContent)
	assertStatus(t, perform(t, app, http.MethodDelete, "/api/v1/access-tokens/"+readID, nil, editorHeaders), fiber.StatusNoContent)
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/v1/queries", nil, workspaceHeaders(readHeaders, "afl")), fiber.StatusUnauthorized, "access_token_revoked")

	// Истечение срока и неизвестный токен.
	if _, err := database.Pool.Exec(context.Background(), `UPDATE access_tokens SET created_at=NOW()-interval '2 hours',expires_at=NOW()-interval '1 hour' WHERE id=$1`, writeID); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/v1/queries", nil, writeHeaders), fiber.StatusUnauthorized, "access_token_expired")
	unknown := bearer("endge_pat_" + strings.Repeat("A", 43))
	assertErrorCode(t, perform(t, app, http.MethodGet, "/api/session/me", nil, unknown), fiber.StatusUnauthorized, "access_token_invalid")

	// Флаг commit выпускается только Admin workspace; удаление разрешается флагом delete.
	assertErrorCode(t, perform(t, app, http.MethodPost, "/api/v1/access-tokens", map[string]any{"workspaceIdentity": "afl", "name": "Agent commit", "scope": "write", "commit": true, "expiresAt": expiresAt}, editorHeaders), fiber.StatusForbidden, "access_token_owner_admin_required")
	fullSecret, _ := createAccessToken(t, app, editorHeaders, map[string]any{"workspaceIdentity": "afl", "name": "Agent full", "scope": "write", "delete": true, "expiresAt": expiresAt}, "write")
	fullHeaders := workspaceHeaders(bearer(fullSecret), "afl")
	deleted := perform(t, app, http.MethodDelete, "/api/v1/queries/agent-query", nil, fullHeaders)
	if deleted.StatusCode == fiber.StatusForbidden {
		t.Fatalf("delete flag must allow deletion")
	}
	deleted.Body.Close()
}

// createAccessToken выпускает токен и проверяет, что секрет возвращён один раз с узнаваемым префиксом.
func createAccessToken(t *testing.T, app *fiber.App, headers map[string]string, body map[string]any, expectedScope string) (string, string) {
	t.Helper()
	response := perform(t, app, http.MethodPost, "/api/v1/access-tokens", body, headers)
	assertStatus(t, response, fiber.StatusCreated)
	if response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
		t.Fatal("token response must not be cached")
	}
	payload := decodeObject(t, response)
	secret := stringField(t, payload, "token")
	token := objectField(t, payload, "accessToken")
	if !strings.HasPrefix(secret, "endge_pat_") || !strings.HasPrefix(secret, stringField(t, token, "tokenPrefix")) {
		t.Fatalf("unexpected token payload")
	}
	if stringField(t, token, "scope") != expectedScope || stringField(t, token, "status") != "active" {
		t.Fatalf("unexpected token metadata: %v", token)
	}
	return secret, stringField(t, token, "id")
}

func assertErrorCode(t *testing.T, response *http.Response, status int, code string) {
	t.Helper()
	if response.StatusCode != status {
		assertStatus(t, response, status)
	}
	if actual := stringField(t, decodeObject(t, response), "code"); actual != code {
		t.Fatalf("code=%q, ожидался %q", actual, code)
	}
}
