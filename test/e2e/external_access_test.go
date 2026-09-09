//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/test/support"
)

const externalAccessYAML = `version: 1
adapter: oidc
provider: test-oidc
displayName: Test external provider
claimsSource: access_token
rules:
  - id: platform
    when: {path: /roles, contains: platform-admin}
    grant: {scope: platform, role: admin}
  - id: admin
    when: {path: /roles, contains: admin}
    grant: {scope: workspace, workspace: default, role: admin}
  - id: editor
    when: {path: /roles, contains: editor}
    grant: {scope: workspace, workspace: default, role: editor}
  - id: viewer
    when: {path: /roles, contains: viewer}
    grant: {scope: workspace, workspace: default, role: viewer}
  - id: missing
    when: {path: /roles, contains: missing}
    grant: {scope: workspace, workspace: does-not-exist, role: viewer}
`

func externalConfig(t *testing.T, provider *support.IdentityProvider, source string) *config.Config {
	t.Helper()
	cfg := support.OIDCConfig(provider)
	var err error
	cfg.Access, err = config.ParseAccessConfig([]byte(source), cfg.Identity, cfg.ConfiguratorAuth)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func externalTokenInput(subject string, issued time.Time, roles ...string) support.TokenInput {
	return support.TokenInput{Subject: subject, Username: subject, IssuedAt: issued, Claims: map[string]any{"roles": roles}}
}
func externalHeaders(t *testing.T, provider *support.IdentityProvider, input support.TokenInput) map[string]string {
	t.Helper()
	return map[string]string{"Authorization": "Bearer " + provider.Token(t, input), "X-Endge-Workspace": "default"}
}
func expectCode(t *testing.T, response *http.Response, status int, code string) {
	t.Helper()
	assertStatus(t, response, status)
	if got := stringField(t, decodeObject(t, response), "code"); got != code {
		t.Fatalf("code=%q want=%q", got, code)
	}
}

func TestExternalAccessBearerLifecycle(t *testing.T) {
	db := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, db, externalConfig(t, provider, externalAccessYAML))
	issued := time.Now().Add(-time.Minute).Truncate(time.Second)
	// First human and legacy admin groups cannot bypass the mapping.
	outsider := externalTokenInput("external-outsider", issued)
	outsider.Groups = []string{"endge-platform-admins"}
	response := perform(t, app, http.MethodGet, "/api/session/me", nil, externalHeaders(t, provider, outsider))
	assertStatus(t, response, 200)
	session := decodeObject(t, response)
	if session["platformAdmin"] != false || len(session["workspaces"].([]any)) != 0 {
		t.Fatalf("unexpected bootstrap: %v", session)
	}
	if objectField(t, session, "accessManagement")["mode"] != "external" {
		t.Fatal("missing external mode")
	}
	for _, role := range []string{"viewer", "editor", "admin", "platform-admin"} {
		header := externalHeaders(t, provider, externalTokenInput("role-"+role, issued, role))
		response = perform(t, app, http.MethodGet, "/api/session/me", nil, header)
		assertStatus(t, response, 200)
		value := decodeObject(t, response)
		if value["platformAdmin"] != (role == "platform-admin") {
			t.Fatalf("wrong platform role for %s", role)
		}
		items := value["workspaces"].([]any)
		if len(items) == 0 {
			t.Fatalf("no workspace for %s", role)
		}
		expected := role
		if role == "platform-admin" {
			expected = "admin"
		}
		if items[0].(map[string]any)["role"] != expected {
			t.Fatalf("wrong workspace role for %s: %v", role, items)
		}
	}
	admin := externalHeaders(t, provider, externalTokenInput("managed", issued, "platform-admin", "admin"))
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 200)
	current := decodeObject(t, response)
	userID := stringField(t, objectField(t, current, "user"), "id")
	management := objectField(t, current, "accessManagement")
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 200)
	if objectField(t, decodeObject(t, response), "accessManagement")["lastSynchronizedAt"] != management["lastSynchronizedAt"] {
		t.Fatal("same token rewrote assignments")
	}
	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "/api/v1/access-grants", map[string]any{"userId": userID, "scopeType": "platform", "role": "admin"}},
		{http.MethodDelete, "/api/v1/access-grants/00000000-0000-0000-0000-000000000123", nil},
		{http.MethodPost, "/api/v1/access-grants/bulk-workspaces", map[string]any{"userId": userID, "role": "admin", "selection": map[string]any{"type": "all-active"}}},
		{http.MethodPut, "/api/v1/workspaces/default/members/" + userID, map[string]any{"role": "admin"}},
		{http.MethodDelete, "/api/v1/workspaces/default/members/" + userID, nil},
	} {
		t.Run(request.method+request.path, func(t *testing.T) {
			expectCode(t, perform(t, app, request.method, request.path, request.body, admin), 403, "access_managed_externally")
		})
	}
	viewer := externalHeaders(t, provider, externalTokenInput("managed", issued.Add(time.Second), "viewer"))
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, viewer)
	assertStatus(t, response, 200)
	value := decodeObject(t, response)
	if value["platformAdmin"] != false || value["workspaces"].([]any)[0].(map[string]any)["role"] != "viewer" {
		t.Fatal("downgrade not applied")
	}
	expectCode(t, perform(t, app, http.MethodGet, "/api/session/me", nil, admin), 403, "external_access_stale")
	ambiguous := externalHeaders(t, provider, externalTokenInput("managed", issued.Add(time.Second), "admin"))
	expectCode(t, perform(t, app, http.MethodGet, "/api/session/me", nil, ambiguous), 403, "external_access_ambiguous")
	// A bad target rolls back the whole set, including removal of prior grants.
	bad := externalHeaders(t, provider, externalTokenInput("managed", issued.Add(2*time.Second), "platform-admin", "missing"))
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, bad)
	assertStatus(t, response, 500)
	response.Body.Close()
	var role string
	if err := db.Pool.QueryRow(t.Context(), `SELECT role FROM access_grants WHERE user_id=$1 AND scope_type='workspace'`, userID).Scan(&role); err != nil || role != "viewer" {
		t.Fatalf("partial mapping persisted: role=%s err=%v", role, err)
	}
	empty := externalHeaders(t, provider, externalTokenInput("managed", issued.Add(3*time.Second)))
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, empty)
	assertStatus(t, response, 200)
	value = decodeObject(t, response)
	if value["platformAdmin"] != false || len(value["workspaces"].([]any)) != 0 {
		t.Fatal("empty set did not revoke all grants")
	}
	var grants, memberships int
	if err := db.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM access_grants WHERE user_id=$1),(SELECT count(*) FROM workspace_memberships WHERE user_id=$1)`, userID).Scan(&grants, &memberships); err != nil || grants != 0 || memberships != 0 {
		t.Fatalf("grants=%d memberships=%d err=%v", grants, memberships, err)
	}
}

func TestExternalAccessBrowserRefreshAndConfigurationChange(t *testing.T) {
	db := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	cfg := externalConfig(t, provider, externalAccessYAML)
	app := support.NewTestApp(t, db, cfg)
	issued := time.Now().Add(-time.Minute).Truncate(time.Second)
	state, nonce, transaction := beginBrowserLogin(t, app)
	identity := externalTokenInput("browser-external", issued, "platform-admin")
	access := externalTokenInput("browser-external", issued, "editor")
	code := provider.AuthorizationCodeWithTokens(t, identity, access, nonce)
	callback := perform(t, app, http.MethodGet, callbackURL(state, code), nil, map[string]string{"Cookie": transaction.Name + "=" + transaction.Value})
	assertStatus(t, callback, 303)
	cookie := responseCookie(t, callback, "endge_test_session")
	callback.Body.Close()
	// Synchronization must already have happened before the browser receives its cookie.
	var role, userID string
	if err := db.Pool.QueryRow(t.Context(), `SELECT g.role,u.id::text FROM access_grants g JOIN service_users u ON u.id=g.user_id WHERE u.subject='browser-external'`).Scan(&role, &userID); err != nil || role != "editor" {
		t.Fatalf("callback mapping: %s %v", role, err)
	}
	headers := map[string]string{"Cookie": cookie.Name + "=" + cookie.Value, "X-Endge-Workspace": "default"}
	refresh := externalTokenInput("browser-external", issued.Add(time.Second), "viewer")
	provider.SetRefreshTokens(t, identity, refresh)
	if _, err := db.Pool.Exec(t.Context(), `UPDATE configurator_auth_sessions SET identity_refresh_at=NOW() WHERE subject='browser-external'`); err != nil {
		t.Fatal(err)
	}
	response := perform(t, app, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, 200)
	value := decodeObject(t, response)
	if value["platformAdmin"] != false || value["workspaces"].([]any)[0].(map[string]any)["role"] != "viewer" {
		t.Fatalf("refresh mapped ID token instead of access token: %v", value)
	}
	if provider.RefreshCalls() != 1 {
		t.Fatalf("refresh calls=%d", provider.RefreshCalls())
	}
	// A new process policy forces refresh even though the stored session is still fresh.
	changed := externalConfig(t, provider, strings.Replace(externalAccessYAML, "role: viewer}", "role: editor}", 1))
	newApp := support.NewTestApp(t, db, changed)
	provider.SetRefreshTokens(t, identity, externalTokenInput("browser-external", issued.Add(2*time.Second), "viewer"))
	response = perform(t, newApp, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, 200)
	value = decodeObject(t, response)
	if value["workspaces"].([]any)[0].(map[string]any)["role"] != "editor" || provider.RefreshCalls() != 2 {
		t.Fatal("configuration change did not refresh and remap")
	}
	provider.SetRefreshTokens(t, identity, externalTokenInput("browser-external", issued.Add(3*time.Second)))
	if _, err := db.Pool.Exec(t.Context(), `UPDATE configurator_auth_sessions SET identity_refresh_at=NOW() WHERE subject='browser-external'`); err != nil {
		t.Fatal(err)
	}
	response = perform(t, newApp, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, 200)
	if len(decodeObject(t, response)["workspaces"].([]any)) != 0 {
		t.Fatal("refresh without roles kept access")
	}
	// Provider identity mismatch revokes the cookie session rather than using its last projection.
	provider.SetRefreshTokens(t, identity, externalTokenInput("someone-else", issued.Add(4*time.Second), "admin"))
	if _, err := db.Pool.Exec(t.Context(), `UPDATE configurator_auth_sessions SET identity_refresh_at=NOW() WHERE subject='browser-external'`); err != nil {
		t.Fatal(err)
	}
	response = perform(t, newApp, http.MethodGet, "/api/session/me", nil, headers)
	assertStatus(t, response, 401)
	response.Body.Close()
}

func TestExternalAccessConcurrentTokens(t *testing.T) {
	db := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, db, externalConfig(t, provider, externalAccessYAML))
	issued := time.Now().Add(-time.Minute).Truncate(time.Second)
	headers := []map[string]string{
		externalHeaders(t, provider, externalTokenInput("racing", issued, "admin")),
		externalHeaders(t, provider, externalTokenInput("racing", issued.Add(time.Second), "viewer")),
	}
	var wg sync.WaitGroup
	errors := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			request, _ := http.NewRequest(http.MethodGet, "http://backend.test/api/session/me", nil)
			for k, v := range headers[i%2] {
				request.Header.Set(k, v)
			}
			response, err := app.Test(request, 10000)
			if err != nil {
				errors <- err
				return
			}
			defer response.Body.Close()
			if response.StatusCode != 200 && response.StatusCode != 403 {
				errors <- fmt.Errorf("concurrent status=%d", response.StatusCode)
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	var roles []byte
	if err := db.Pool.QueryRow(t.Context(), `SELECT jsonb_agg(g.role) FROM access_grants g JOIN service_users u ON u.id=g.user_id WHERE u.subject='racing'`).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	var values []string
	if err := json.Unmarshal(roles, &values); err != nil || len(values) != 1 || values[0] != "viewer" {
		t.Fatalf("old concurrent token won: %s %v", roles, err)
	}
}

func TestExternalAccessModeTransition(t *testing.T) {
	db := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	external := support.NewTestApp(t, db, externalConfig(t, provider, externalAccessYAML))
	issued := time.Now().Add(-time.Minute)
	empty := externalHeaders(t, provider, externalTokenInput("switch-user", issued))
	response := perform(t, external, http.MethodGet, "/api/session/me", nil, empty)
	assertStatus(t, response, 200)
	response.Body.Close()
	local := support.NewTestApp(t, db, support.OIDCConfig(provider))
	response = perform(t, local, http.MethodGet, "/api/session/me", nil, empty)
	assertStatus(t, response, 200)
	if decodeObject(t, response)["platformAdmin"] != false {
		t.Fatal("return to local bootstrapped an emergency admin")
	}
	admin := externalHeaders(t, provider, externalTokenInput("switch-user", issued.Add(time.Second), "platform-admin", "viewer"))
	response = perform(t, external, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 200)
	userID := stringField(t, objectField(t, decodeObject(t, response), "user"), "id")
	response = perform(t, local, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 200)
	value := decodeObject(t, response)
	if value["platformAdmin"] != true || objectField(t, value, "accessManagement")["mode"] != "local" {
		t.Fatal("local mode did not preserve the last external snapshot")
	}
	response = perform(t, local, http.MethodPut, "/api/v1/workspaces/default/members/"+userID, map[string]any{"role": "editor"}, admin)
	assertStatus(t, response, 200)
	response.Body.Close()
	// Re-enter external mode with the same token. Its old watermark must not hide the local edit.
	response = perform(t, external, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 200)
	response.Body.Close()
	var role string
	if err := db.Pool.QueryRow(t.Context(), `SELECT role FROM access_grants WHERE user_id=$1 AND scope_type='workspace'`, userID).Scan(&role); err != nil || role != "viewer" {
		t.Fatalf("local override survived return to external: %s %v", role, err)
	}
}

func TestExternalAccessTransactionRollsBackWriteFailure(t *testing.T) {
	db := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, db, externalConfig(t, provider, externalAccessYAML))
	issued := time.Now().Add(-time.Minute)
	viewer := externalHeaders(t, provider, externalTokenInput("rollback-user", issued, "viewer"))
	response := perform(t, app, http.MethodGet, "/api/session/me", nil, viewer)
	assertStatus(t, response, 200)
	value := decodeObject(t, response)
	before := objectField(t, value, "accessManagement")["lastSynchronizedAt"]
	if _, err := db.Pool.Exec(t.Context(), `CREATE FUNCTION reject_external_admin() RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN IF NEW.scope_type='workspace' AND NEW.role='admin' THEN RAISE EXCEPTION 'injected write failure'; END IF; RETURN NEW; END $$;
	CREATE TRIGGER reject_external_admin BEFORE INSERT ON access_grants FOR EACH ROW EXECUTE FUNCTION reject_external_admin();`); err != nil {
		t.Fatal(err)
	}
	admin := externalHeaders(t, provider, externalTokenInput("rollback-user", issued.Add(time.Second), "platform-admin", "admin"))
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, admin)
	assertStatus(t, response, 500)
	response.Body.Close()
	response = perform(t, app, http.MethodGet, "/api/session/me", nil, viewer)
	assertStatus(t, response, 200)
	value = decodeObject(t, response)
	if value["platformAdmin"] != false || value["workspaces"].([]any)[0].(map[string]any)["role"] != "viewer" || objectField(t, value, "accessManagement")["lastSynchronizedAt"] != before {
		t.Fatal("failed transaction changed grants or watermark")
	}
}
