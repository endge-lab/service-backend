package entities

import (
	"testing"
	"time"
)

func TestAccessTokenCapsWorkspaceRole(t *testing.T) {
	read := AccessTokenPrincipal{Scope: AccessTokenScopeRead}
	write := AccessTokenPrincipal{Scope: AccessTokenScopeWrite}
	cases := []struct {
		principal AccessTokenPrincipal
		role      string
		expected  string
	}{
		{read, "", ""},
		{read, "viewer", "viewer"},
		{read, "editor", "viewer"},
		{read, "platform_admin", "viewer"},
		{write, "viewer", "viewer"},
		{write, "editor", "editor"},
		{write, "admin", "editor"},
		{write, "platform_admin", "editor"},
	}
	for _, tc := range cases {
		if actual := tc.principal.CapWorkspaceRole(tc.role); actual != tc.expected {
			t.Fatalf("%s token with role %q: expected %q, got %q", tc.principal.Scope, tc.role, tc.expected, actual)
		}
	}
}

func TestAccessTokenStatusPrefersRevocation(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	token := AccessToken{ExpiresAt: now.Add(time.Minute)}
	if token.Status(now) != AccessTokenStatusActive {
		t.Fatal("token must be active before expiry")
	}
	token.ExpiresAt = now
	if token.Status(now) != AccessTokenStatusExpired {
		t.Fatal("token must expire exactly at expires_at")
	}
	token.RevokedAt = &now
	if token.Status(now) != AccessTokenStatusRevoked {
		t.Fatal("revocation must win over expiry")
	}
}
