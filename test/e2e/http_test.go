//go:build e2e

package e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

type documentHTTPCase struct {
	collection string
	identity   string
	payload    map[string]any
}

// TestOIDCCurrentUserAndProtectedRoutes проверяет весь bearer -> local user -> session pipeline.
func TestOIDCCurrentUserAndProtectedRoutes(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))

	public := perform(t, app, http.MethodGet, "/health", nil, nil)
	assertStatus(t, public, fiber.StatusOK)
	unauthorized := perform(t, app, http.MethodGet, "/api/session/me", nil, nil)
	assertStatus(t, unauthorized, fiber.StatusUnauthorized)

	token := provider.Token(t, support.TokenInput{Subject: "oidc-user", Username: "alice", DisplayName: "Alice", Groups: []string{"endge-platform-admins"}})
	headers := map[string]string{"Authorization": "Bearer " + token}
	first := perform(t, app, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, first, fiber.StatusOK)
	firstBody := decodeObject(t, first)
	user := objectField(t, firstBody, "user")
	userID := stringField(t, user, "id")
	if userID == "" || stringField(t, user, "username") != "alice" {
		t.Fatalf("неверная local user projection: %#v", user)
	}
	second := perform(t, app, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, second, fiber.StatusOK)
	if got := stringField(t, objectField(t, decodeObject(t, second), "user"), "id"); got != userID {
		t.Fatalf("local UUID изменился: first=%s second=%s", userID, got)
	}
	var count int
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM service_users WHERE provider_id='test-oidc' AND subject='oidc-user'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("local user rows=%d err=%v", count, err)
	}

	updatedToken := provider.Token(t, support.TokenInput{Subject: "oidc-user", Username: "alice-renamed", DisplayName: "Alice Renamed", Groups: []string{"endge-platform-admins"}})
	updated := perform(t, app, http.MethodGet, "/api/session/me", nil, map[string]string{"Authorization": "Bearer " + updatedToken})
	assertStatus(t, updated, fiber.StatusOK)
	updatedUser := objectField(t, decodeObject(t, updated), "user")
	if stringField(t, updatedUser, "id") != userID || stringField(t, updatedUser, "username") != "alice-renamed" {
		t.Fatalf("профиль пользователя не синхронизирован: %#v", updatedUser)
	}
	if _, err := database.Pool.Exec(t.Context(), `UPDATE service_users SET active=FALSE WHERE id=$1`, userID); err != nil {
		t.Fatalf("деактивировать пользователя: %v", err)
	}
	inactive := perform(t, app, http.MethodGet, "/api/session/me", nil, map[string]string{"Authorization": "Bearer " + updatedToken})
	assertStatus(t, inactive, fiber.StatusForbidden)
}

// TestCookieAuthenticationRequiresAllowedOrigin проверяет CSRF-защиту изменяющих cookie-запросов.
func TestCookieAuthenticationRequiresAllowedOrigin(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	cfg := support.DevConfig()
	app := support.NewTestApp(t, database, cfg)
	cookieToken := "e2e-cookie-token"
	tokenHash := sha256.Sum256([]byte(cookieToken))
	_, err := database.Pool.Exec(t.Context(), `
		INSERT INTO configurator_auth_sessions(
			token_hash,provider_id,subject,issuer,username,display_name,groups_json,platform_admin,
			identity_refresh_at,expires_at)
		VALUES($1,'cookie-test','cookie-user','urn:endge:test','cookie-user','Cookie User','[]',TRUE,$2,$3)`,
		tokenHash[:], time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	if err != nil {
		t.Fatalf("создать тестовую browser session: %v", err)
	}
	cookieHeaders := map[string]string{"Cookie": cfg.ConfiguratorAuth.SessionCookieName + "=" + cookieToken, "X-Endge-Workspace": "default"}
	read := perform(t, app, http.MethodGet, "/api/v1/queries", nil, cookieHeaders)
	assertStatus(t, read, fiber.StatusOK)

	withoutOrigin := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "csrf-no-origin", "displayName": "Rejected", "source": "query {}", "sourceVersion": 2}, cookieHeaders)
	assertStatus(t, withoutOrigin, fiber.StatusForbidden)
	badOrigin := cloneHeaders(cookieHeaders)
	badOrigin["Origin"] = "https://attacker.example"
	rejected := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "csrf-bad-origin", "displayName": "Rejected", "source": "query {}", "sourceVersion": 2}, badOrigin)
	assertStatus(t, rejected, fiber.StatusForbidden)
	allowedOrigin := cloneHeaders(cookieHeaders)
	allowedOrigin["Origin"] = "http://configurator.test"
	accepted := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "csrf-allowed", "displayName": "Allowed", "source": "query {}", "sourceVersion": 2}, allowedOrigin)
	assertStatus(t, accepted, fiber.StatusCreated)
}

// TestAllDocumentHTTPContracts проверяет CRUD, validation, ETag и soft-delete всех коллекций.
func TestAllDocumentHTTPContracts(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	headers := map[string]string{"X-Endge-Workspace": "default"}

	createdETags := map[string]string{}
	for _, testCase := range documentHTTPCases() {
		t.Run("создание "+testCase.collection, func(t *testing.T) {
			response := perform(t, app, http.MethodPost, "/api/v1/"+testCase.collection, testCase.payload, headers)
			assertStatus(t, response, fiber.StatusCreated)
			etag := response.Header.Get("ETag")
			if etag != `"1"` {
				t.Fatalf("create ETag=%q, ожидался \"1\"", etag)
			}
			body := decodeObject(t, response)
			if stringField(t, body, "identity") != testCase.identity || numberField(t, body, "revision") != 1 {
				t.Fatalf("неверный create response: %#v", body)
			}
			if objectField(t, body, "createdBy")["id"] == nil || objectField(t, body, "updatedBy")["id"] == nil {
				t.Fatalf("audit actor отсутствует: %#v", body)
			}
			createdETags[testCase.collection] = etag
		})
	}

	for _, testCase := range documentHTTPCases() {
		t.Run("полный lifecycle "+testCase.collection, func(t *testing.T) {
			baseURL := "/api/v1/" + testCase.collection + "/" + testCase.identity
			get := perform(t, app, http.MethodGet, baseURL, nil, headers)
			assertStatus(t, get, fiber.StatusOK)
			list := perform(t, app, http.MethodGet, "/api/v1/"+testCase.collection+"?limit=1&offset=0&active=true", nil, headers)
			assertStatus(t, list, fiber.StatusOK)

			missing := perform(t, app, http.MethodPatch, baseURL, map[string]any{"displayName": testCase.identity + " updated"}, headers)
			assertStatus(t, missing, fiber.StatusPreconditionRequired)

			patchHeaders := cloneHeaders(headers)
			patchHeaders["If-Match"] = createdETags[testCase.collection]
			noOp := perform(t, app, http.MethodPatch, baseURL, map[string]any{"displayName": testCase.identity}, patchHeaders)
			assertStatus(t, noOp, fiber.StatusOK)
			if noOp.Header.Get("ETag") != `"1"` {
				t.Fatalf("no-op изменил ETag: %q", noOp.Header.Get("ETag"))
			}
			changed := perform(t, app, http.MethodPatch, baseURL, map[string]any{"displayName": testCase.identity + " updated"}, patchHeaders)
			assertStatus(t, changed, fiber.StatusOK)
			if changed.Header.Get("ETag") != `"2"` {
				t.Fatalf("patch ETag=%q, ожидался \"2\"", changed.Header.Get("ETag"))
			}
			stale := perform(t, app, http.MethodPatch, baseURL, map[string]any{"displayName": "stale"}, patchHeaders)
			assertStatus(t, stale, fiber.StatusConflict)

			deleteHeaders := cloneHeaders(headers)
			deleteHeaders["If-Match"] = `"2"`
			deleted := perform(t, app, http.MethodDelete, baseURL, nil, deleteHeaders)
			assertStatus(t, deleted, fiber.StatusOK)
			if deleted.Header.Get("ETag") != `"3"` {
				t.Fatalf("delete ETag=%q, ожидался \"3\"", deleted.Header.Get("ETag"))
			}
			hidden := perform(t, app, http.MethodGet, baseURL, nil, headers)
			assertStatus(t, hidden, fiber.StatusNotFound)
			included := perform(t, app, http.MethodGet, baseURL+"?includeDeleted=true", nil, headers)
			assertStatus(t, included, fiber.StatusOK)
			included.Body.Close()
			if testCase.collection == "queries" {
				live := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain", nil, headers))
				assertPortableDocumentDeletedState(t, live, "queries", testCase.identity, true)
				exported := decodeObject(t, perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, headers))
				assertPortableDocumentDeletedState(t, exported, "queries", testCase.identity, false)
			}

			restoreHeaders := cloneHeaders(headers)
			restoreHeaders["If-Match"] = `"3"`
			restored := perform(t, app, http.MethodPost, baseURL+"/restore", nil, restoreHeaders)
			assertStatus(t, restored, fiber.StatusOK)
			if restored.Header.Get("ETag") != `"4"` {
				t.Fatalf("restore ETag=%q, ожидался \"4\"", restored.Header.Get("ETag"))
			}
		})
	}
}

func assertPortableDocumentDeletedState(t *testing.T, bundle map[string]any, collection, identity string, expected bool) {
	t.Helper()
	documents, _ := bundle["documents"].(map[string]any)
	items, _ := documents[collection].([]any)
	found := false
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if fmt.Sprint(item["identity"]) != identity {
			continue
		}
		state, _ := item["state"].(map[string]any)
		found = state != nil && state["deletedAt"] != nil
	}
	if found != expected {
		t.Fatalf("documents.%s/%s deleted state=%t, ожидалось %t", collection, identity, found, expected)
	}
}

// TestHTTPValidationAndConcurrency проверяет malformed input, secrets и конкурентный PATCH.
func TestHTTPValidationAndConcurrency(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	headers := map[string]string{"X-Endge-Workspace": "default"}

	invalidBodies := []struct {
		name string
		body string
	}{
		{name: "неизвестное поле", body: `{"identity":"bad","displayName":"Bad","source":"query {}","sourceVersion":2,"legacy":true}`},
		{name: "оборванный JSON", body: `{"identity":`},
		{name: "два JSON значения", body: `{"identity":"bad"}{}`},
		{name: "неверная версия Query", body: `{"identity":"bad","displayName":"Bad","source":"query {}","sourceVersion":1}`},
		{name: "слишком длинный identity", body: `{"identity":"` + strings.Repeat("a", 161) + `","displayName":"Bad","source":"query {}","sourceVersion":2}`},
		{name: "неверный managedBy", body: `{"identity":"bad","displayName":"Bad","managedBy":"attacker","source":"query {}","sourceVersion":2}`},
		{name: "source не строка", body: `{"identity":"bad","displayName":"Bad","source":{"nested":true},"sourceVersion":2}`},
		{name: "подмена actor", body: `{"identity":"bad","displayName":"Bad","source":"query {}","sourceVersion":2,"createdBy":{"id":"attacker"}}`},
		{name: "вложенный secret", body: `{"identity":"bad","displayName":"Bad","source":"query {}","sourceVersion":2,"meta":{"nested":{"clientSecret":"value"}}}`},
	}
	for _, testCase := range invalidBodies {
		t.Run(testCase.name, func(t *testing.T) {
			response := performRaw(t, app, http.MethodPost, "/api/v1/queries", testCase.body, headers)
			if response.StatusCode < 400 || response.StatusCode >= 500 {
				t.Fatalf("невалидный body вернул status=%d", response.StatusCode)
			}
			body := decodeObject(t, response)
			if strings.TrimSpace(fmt.Sprint(body["code"])) == "" || strings.TrimSpace(fmt.Sprint(body["message"])) == "" {
				t.Fatalf("ошибка не соответствует envelope: %#v", body)
			}
		})
	}

	create := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "concurrent", "displayName": "Concurrent", "source": "query {}", "sourceVersion": 2}, headers)
	assertStatus(t, create, fiber.StatusCreated)
	concurrentHeaders := cloneHeaders(headers)
	concurrentHeaders["If-Match"] = `"1"`
	first := perform(t, app, http.MethodPatch, "/api/v1/queries/concurrent", map[string]any{"displayName": "First"}, concurrentHeaders)
	assertStatus(t, first, fiber.StatusOK)
	second := perform(t, app, http.MethodPatch, "/api/v1/queries/concurrent", map[string]any{"displayName": "Second"}, concurrentHeaders)
	assertStatus(t, second, fiber.StatusConflict)
}

// TestFolderSafetyAndReparenting проверяет неизменяемые roots, циклы и атомарный перенос содержимого.
func TestFolderSafetyAndReparenting(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	headers := map[string]string{"X-Endge-Workspace": "default"}

	root := perform(t, app, http.MethodGet, "/api/v1/folders/root-queries", nil, headers)
	assertStatus(t, root, fiber.StatusOK)
	rootETag := root.Header.Get("ETag")
	root.Body.Close()
	rootDeleteHeaders := cloneHeaders(headers)
	rootDeleteHeaders["If-Match"] = rootETag
	rootDelete := perform(t, app, http.MethodDelete, "/api/v1/folders/root-queries", nil, rootDeleteHeaders)
	assertStatus(t, rootDelete, fiber.StatusConflict)

	parent := perform(t, app, http.MethodPost, "/api/v1/folders", map[string]any{"identity": "folder-parent", "displayName": "Parent", "entityType": "queries"}, headers)
	assertStatus(t, parent, fiber.StatusCreated)
	parent.Body.Close()
	child := perform(t, app, http.MethodPost, "/api/v1/folders", map[string]any{"identity": "folder-child", "displayName": "Child", "entityType": "queries", "parentIdentity": "folder-parent"}, headers)
	assertStatus(t, child, fiber.StatusCreated)
	child.Body.Close()
	query := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "folder-query", "displayName": "Folder Query", "folderIdentity": "folder-parent", "source": "query {}", "sourceVersion": 2}, headers)
	assertStatus(t, query, fiber.StatusCreated)
	query.Body.Close()

	selfHeaders := cloneHeaders(headers)
	selfHeaders["If-Match"] = `"1"`
	selfParent := perform(t, app, http.MethodPatch, "/api/v1/folders/folder-parent", map[string]any{"parentIdentity": "folder-parent"}, selfHeaders)
	assertStatus(t, selfParent, fiber.StatusBadRequest)
	cycle := perform(t, app, http.MethodPatch, "/api/v1/folders/folder-parent", map[string]any{"parentIdentity": "folder-child"}, selfHeaders)
	assertStatus(t, cycle, fiber.StatusBadRequest)

	deleted := perform(t, app, http.MethodDelete, "/api/v1/folders/folder-parent", nil, selfHeaders)
	assertStatus(t, deleted, fiber.StatusOK)
	deleted.Body.Close()
	movedChild := perform(t, app, http.MethodGet, "/api/v1/folders/folder-child", nil, headers)
	assertStatus(t, movedChild, fiber.StatusOK)
	childBody := decodeObject(t, movedChild)
	if stringField(t, childBody, "parentIdentity") != "root-queries" || numberField(t, childBody, "revision") != 2 {
		t.Fatalf("дочерняя папка не перенесена в root: %#v", childBody)
	}
	movedQuery := perform(t, app, http.MethodGet, "/api/v1/queries/folder-query", nil, headers)
	assertStatus(t, movedQuery, fiber.StatusOK)
	queryBody := decodeObject(t, movedQuery)
	if stringField(t, queryBody, "folderIdentity") != "root-queries" || numberField(t, queryBody, "revision") != 2 {
		t.Fatalf("документ не перенесён в root: %#v", queryBody)
	}
}

// TestBulkDocumentMove проверяет один атомарный запрос, optimistic revisions и общую папку назначения.
func TestBulkDocumentMove(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	headers := map[string]string{"X-Endge-Workspace": "default"}

	folder := perform(t, app, http.MethodPost, "/api/v1/folders", map[string]any{
		"identity": "schedule-actions", "displayName": "Schedule", "entityType": "actions",
	}, headers)
	assertStatus(t, folder, fiber.StatusCreated)
	folder.Body.Close()
	for _, identity := range []string{"action-a", "action-b"} {
		created := perform(t, app, http.MethodPost, "/api/v1/actions", map[string]any{
			"identity": identity, "displayName": identity,
		}, headers)
		assertStatus(t, created, fiber.StatusCreated)
		created.Body.Close()
	}

	move := perform(t, app, http.MethodPost, "/api/v1/domain/documents/move", map[string]any{
		"folderIdentity": "schedule-actions",
		"documents": []map[string]any{
			{"collection": "actions", "identity": "action-a", "expectedRevision": 1},
			{"collection": "actions", "identity": "action-b", "expectedRevision": 1},
		},
	}, headers)
	assertStatus(t, move, fiber.StatusOK)
	moveBody := decodeObject(t, move)
	if numberField(t, moveBody, "moved") != 2 {
		t.Fatalf("неверное число перемещённых документов: %#v", moveBody)
	}
	items, ok := moveBody["documents"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("неверный bulk move response: %#v", moveBody)
	}

	conflict := perform(t, app, http.MethodPost, "/api/v1/domain/documents/move", map[string]any{
		"folderIdentity": "root-actions",
		"documents": []map[string]any{
			{"collection": "actions", "identity": "action-a", "expectedRevision": 2},
			{"collection": "actions", "identity": "action-b", "expectedRevision": 1},
		},
	}, headers)
	assertStatus(t, conflict, fiber.StatusConflict)
	conflict.Body.Close()

	for _, identity := range []string{"action-a", "action-b"} {
		actual := perform(t, app, http.MethodGet, "/api/v1/actions/"+identity, nil, headers)
		assertStatus(t, actual, fiber.StatusOK)
		body := decodeObject(t, actual)
		if stringField(t, body, "folderIdentity") != "schedule-actions" || numberField(t, body, "revision") != 2 {
			t.Fatalf("атомарность bulk move нарушена для %s: %#v", identity, body)
		}
	}
}

func documentHTTPCases() []documentHTTPCase {
	base := func(identity string) map[string]any {
		return map[string]any{"identity": identity, "displayName": identity}
	}
	with := func(value map[string]any, pairs ...any) map[string]any {
		for index := 0; index < len(pairs); index += 2 {
			value[fmt.Sprint(pairs[index])] = pairs[index+1]
		}
		return value
	}
	return []documentHTTPCase{
		{collection: "environments", identity: "environment-main", payload: base("environment-main")},
		{collection: "stores", identity: "store-main", payload: with(base("store-main"), "source", "store {}", "sourceVersion", 1)},
		{collection: "auth-profiles", identity: "auth-main", payload: with(base("auth-main"), "adapterId", "oidc", "config", map[string]any{}, "credentialRefs", map[string]any{"client": "vault://client"}, "persist", "memory")},
		{collection: "projects", identity: "project-main", payload: with(base("project-main"), "allowedEnvironments", []any{"environment-main"})},
		{collection: "tenants", identity: "tenant-main", payload: with(base("tenant-main"), "code", "TENANT")},
		{collection: "folders", identity: "folder-main", payload: with(base("folder-main"), "entityType", "queries")},
		{collection: "types", identity: "type-main", payload: with(base("type-main"), "source", "type {}", "sourceVersion", 1)},
		{collection: "queries", identity: "query-main", payload: with(base("query-main"), "source", "query {}", "sourceVersion", 2)},
		{collection: "data-views", identity: "data-view-main", payload: with(base("data-view-main"), "source", "view {}", "sourceVersion", 1)},
		{collection: "compositions", identity: "composition-main", payload: with(base("composition-main"), "kind", "screen", "source", "composition {}", "sourceVersion", 1)},
		{collection: "streams", identity: "stream-main", payload: with(base("stream-main"), "source", "stream {}", "sourceVersion", 1)},
		{collection: "updates", identity: "update-main", payload: with(base("update-main"), "storeIdentity", "store-main", "source", "update {}", "sourceVersion", 1)},
		{collection: "mocks", identity: "mock-main", payload: with(base("mock-main"), "contentSource", "inline", "contentType", "application/json", "source", "{}")},
		{collection: "components", identity: "component-main", payload: with(base("component-main"), "source", "<template />", "tag", "endge-main", "modelVersion", 1, "supportedTargets", []any{"vue"})},
		{collection: "actions", identity: "action-main", payload: with(base("action-main"), "definition", map[string]any{}, "input", map[string]any{}, "output", map[string]any{}, "target", map[string]any{})},
		{collection: "filters", identity: "filter-main", payload: with(base("filter-main"), "fields", []any{}, "source", "filter {}", "sourceVersion", 1)},
		{collection: "converters", identity: "converter-main", payload: base("converter-main")},
		{collection: "computations", identity: "computation-main", payload: with(base("computation-main"), "source", "compute {}", "sourceVersion", 1, "contractVersion", 1)},
		{collection: "vocabs", identity: "vocab-main", payload: with(base("vocab-main"), "mode", "external_payload", "authMode", "profile", "authProfileIdentity", "auth-main")},
		{collection: "i18n-bundles", identity: "i18n-main", payload: with(base("i18n-main"), "locales", map[string]any{"ru": map[string]any{"title": "Тест"}})},
		{collection: "navigations", identity: "navigation-main", payload: with(base("navigation-main"), "tree", []any{})},
		{collection: "styles", identity: "style-main", payload: with(base("style-main"), "source", "body {}", "sourceVersion", 1)},
		{collection: "configurations", identity: "configuration-main", payload: with(base("configuration-main"), "source", "defineConfig({ enabled: value(Boolean, true) })", "sourceVersion", 1)},
	}
}

func perform(t *testing.T, app *fiber.App, method, target string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var raw string
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("сериализовать HTTP body: %v", err)
		}
		raw = string(encoded)
	}
	return performRaw(t, app, method, target, raw, headers)
}

func performRaw(t *testing.T, app *fiber.App, method, target, raw string, headers map[string]string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(raw))
	if raw != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("выполнить %s %s: %v", method, target, err)
	}
	return response
}

func assertStatus(t *testing.T, response *http.Response, expected int) {
	t.Helper()
	if response.StatusCode == expected {
		return
	}
	raw, _ := io.ReadAll(response.Body)
	t.Fatalf("status=%d, ожидался %d, body=%s", response.StatusCode, expected, raw)
}

func decodeObject(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	defer response.Body.Close()
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatalf("декодировать HTTP response: %v", err)
	}
	return value
}

func objectField(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	result, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("поле %s не является object: %#v", key, value[key])
	}
	return result
}

func stringField(t *testing.T, value map[string]any, key string) string {
	t.Helper()
	result, ok := value[key].(string)
	if !ok {
		t.Fatalf("поле %s не является string: %#v", key, value[key])
	}
	return result
}

func numberField(t *testing.T, value map[string]any, key string) int {
	t.Helper()
	result, ok := value[key].(float64)
	if !ok {
		t.Fatalf("поле %s не является number: %#v", key, value[key])
	}
	return int(result)
}

func cloneHeaders(value map[string]string) map[string]string {
	result := make(map[string]string, len(value)+1)
	for name, item := range value {
		result[name] = item
	}
	return result
}
