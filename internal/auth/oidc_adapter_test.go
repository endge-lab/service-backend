package auth

import (
	"net/url"
	"testing"

	"github.com/endge-lab/service-backend/internal/config"
)

// TestOIDCAdapterBuildsProviderRedirectWithoutFrontendSDK проверяет PKCE redirect на стороне backend.
func TestOIDCAdapterBuildsProviderRedirectWithoutFrontendSDK(t *testing.T) {
	adapter := NewOIDCAdapter(&config.Config{ConfiguratorAuth: config.ConfiguratorAuthConfig{
		AuthorizationURL: "https://identity.example/authorize",
		ClientID:         "configurator",
		RedirectURL:      "https://backend.example/auth/callback",
		Scopes:           []string{"openid", "email"},
	}})
	raw, err := adapter.LoginURL("state", "challenge", "nonce")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	for key, expected := range map[string]string{
		"response_type": "code", "client_id": "configurator", "redirect_uri": "https://backend.example/auth/callback",
		"scope": "openid email", "state": "state", "nonce": "nonce", "code_challenge": "challenge", "code_challenge_method": "S256",
	} {
		if query.Get(key) != expected {
			t.Fatalf("%s=%q, want %q", key, query.Get(key), expected)
		}
	}
}

// TestSessionManagerRejectsExternalReturnURL проверяет защиту от open redirect.
func TestSessionManagerRejectsExternalReturnURL(t *testing.T) {
	manager := &SessionManager{config: config.ConfiguratorAuthConfig{
		ReturnURL:            "https://configurator.example/start",
		AllowedReturnOrigins: []string{"http://localhost:5173"},
	}}
	if got := manager.safeReturnURL("https://attacker.example/callback"); got != "https://configurator.example/start" {
		t.Fatalf("unsafe return URL accepted: %q", got)
	}
	if got := manager.safeReturnURL("/workspace/default"); got != "https://configurator.example/workspace/default" {
		t.Fatalf("relative return URL resolved to %q", got)
	}
	if got := manager.safeReturnURL("http://localhost:5173/workspace/default?tab=source#editor"); got != "http://localhost:5173/workspace/default?tab=source#editor" {
		t.Fatalf("allowed local return URL rejected: %q", got)
	}
	if got := manager.safeReturnURL("http://localhost:5174/workspace/default"); got != "https://configurator.example/start" {
		t.Fatalf("unlisted local origin accepted: %q", got)
	}
}
