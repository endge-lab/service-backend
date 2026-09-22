//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

const loginTransactionCookieName = "endge_configurator_login"

// TestConfiguratorBrowserAuthFlow verifies the complete browser login flow:
// OIDC redirect, state/nonce binding, callback, server-side session and logout.
func TestConfiguratorBrowserAuthFlow(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))

	providerRejected := perform(t, app, http.MethodGet, "/auth/callback?error=access_denied", nil, nil)
	assertStatus(t, providerRejected, fiber.StatusUnauthorized)
	providerRejected.Body.Close()

	state, nonce, transactionCookie := beginBrowserLogin(t, app)
	code := provider.AuthorizationCode(t, support.TokenInput{
		Subject: "browser-user", Username: "browser-user", DisplayName: "Browser User",
	}, nonce)
	wrongCookieHeaders := map[string]string{"Cookie": loginTransactionCookieName + "=wrong-browser-binding"}
	wrongBinding := perform(t, app, http.MethodGet, callbackURL(state, code), nil, wrongCookieHeaders)
	assertStatus(t, wrongBinding, fiber.StatusUnauthorized)
	wrongBinding.Body.Close()

	callbackHeaders := map[string]string{"Cookie": transactionCookie.Name + "=" + transactionCookie.Value}
	callback := perform(t, app, http.MethodGet, callbackURL(state, code), nil, callbackHeaders)
	assertStatus(t, callback, fiber.StatusSeeOther)
	if callback.Header.Get(fiber.HeaderLocation) != "http://configurator.test/after-login" {
		callback.Body.Close()
		t.Fatalf("callback return location=%q", callback.Header.Get(fiber.HeaderLocation))
	}
	sessionCookie := responseCookie(t, callback, "endge_test_session")
	callback.Body.Close()

	current := perform(t, app, http.MethodGet, "/api/session/me", nil, map[string]string{"Cookie": sessionCookie.Name + "=" + sessionCookie.Value})
	assertStatus(t, current, fiber.StatusOK)
	if stringField(t, objectField(t, decodeObject(t, current), "user"), "username") != "browser-user" {
		t.Fatal("callback session did not expose the OIDC user")
	}
	var activeSessions int
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM configurator_auth_sessions WHERE revoked_at IS NULL`).Scan(&activeSessions); err != nil || activeSessions != 1 {
		t.Fatalf("active browser sessions=%d err=%v", activeSessions, err)
	}

	replayed := perform(t, app, http.MethodGet, callbackURL(state, code), nil, callbackHeaders)
	assertStatus(t, replayed, fiber.StatusUnauthorized)
	replayed.Body.Close()
	invalidState, _, invalidTransactionCookie := beginBrowserLogin(t, app)
	invalidCode := perform(t, app, http.MethodGet, callbackURL(invalidState, "not-issued"), nil, map[string]string{"Cookie": invalidTransactionCookie.Name + "=" + invalidTransactionCookie.Value})
	assertStatus(t, invalidCode, fiber.StatusUnauthorized)
	invalidCode.Body.Close()

	logout := perform(t, app, http.MethodPost, "/auth/logout", nil, map[string]string{"Cookie": sessionCookie.Name + "=" + sessionCookie.Value})
	assertStatus(t, logout, fiber.StatusNoContent)
	clearedCookie := responseCookie(t, logout, sessionCookie.Name)
	if clearedCookie.Value != "" || clearedCookie.MaxAge >= 0 {
		logout.Body.Close()
		t.Fatalf("logout did not clear session cookie: %#v", clearedCookie)
	}
	logout.Body.Close()
	if provider.LogoutCalls() != 1 {
		t.Fatalf("OIDC provider logout calls=%d, expected 1", provider.LogoutCalls())
	}
	var revokedSessions int
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM configurator_auth_sessions WHERE revoked_at IS NOT NULL`).Scan(&revokedSessions); err != nil || revokedSessions != 1 {
		t.Fatalf("revoked browser sessions=%d err=%v", revokedSessions, err)
	}
	afterLogout := perform(t, app, http.MethodGet, "/api/session/me", nil, map[string]string{"Cookie": sessionCookie.Name + "=" + sessionCookie.Value})
	assertStatus(t, afterLogout, fiber.StatusUnauthorized)
	afterLogout.Body.Close()

	emptyLogout := perform(t, app, http.MethodPost, "/auth/logout", nil, nil)
	assertStatus(t, emptyLogout, fiber.StatusNoContent)
	emptyLogout.Body.Close()
}

func beginBrowserLogin(t *testing.T, app *fiber.App) (string, string, *http.Cookie) {
	t.Helper()

	response := perform(t, app, http.MethodGet, "/auth/login?returnTo=/after-login", nil, nil)
	assertStatus(t, response, fiber.StatusFound)
	location, err := url.Parse(response.Header.Get(fiber.HeaderLocation))
	if err != nil {
		response.Body.Close()
		t.Fatalf("parse OIDC login location: %v", err)
	}
	query := location.Query()
	if query.Get("response_type") != "code" || query.Get("code_challenge_method") != "S256" || query.Get("state") == "" || query.Get("nonce") == "" {
		response.Body.Close()
		t.Fatalf("OIDC login redirect misses PKCE/state/nonce: %s", location.String())
	}
	cookie := responseCookie(t, response, loginTransactionCookieName)
	response.Body.Close()
	return query.Get("state"), query.Get("nonce"), cookie
}

func callbackURL(state, code string) string {
	return "/auth/callback?" + url.Values{"state": []string{state}, "code": []string{code}}.Encode()
}

func responseCookie(t *testing.T, response *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("response does not set cookie %q", name)
	return nil
}
