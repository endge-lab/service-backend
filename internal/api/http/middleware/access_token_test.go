package middleware

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

func TestBearerAccessTokenRecognizesOnlyPATPrefix(t *testing.T) {
	if secret, ok := bearerAccessToken("bearer endge_pat_abc"); !ok || secret != "endge_pat_abc" {
		t.Fatalf("PAT must be recognised case-insensitively, got %q %v", secret, ok)
	}
	for _, header := range []string{"", "Bearer eyJhbGciOiJSUzI1NiJ9.x.y", "Basic endge_pat_abc", "endge_pat_abc"} {
		if _, ok := bearerAccessToken(header); ok {
			t.Fatalf("header %q must not be treated as PAT", header)
		}
	}
}

func TestAccessTokenRoutePolicy(t *testing.T) {
	read := entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeRead}
	write := entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite}
	full := entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite, CanCommit: true, CanDelete: true}
	cases := []struct {
		name      string
		principal entities.AccessTokenPrincipal
		method    string
		path      string
		code      string
	}{
		{"read documents", read, "GET", "/api/v1/queries", ""},
		{"session", read, "GET", "/api/session/me", ""},
		{"token management", full, "POST", "/api/v1/access-tokens", "access_token_management_forbidden"},
		{"grants", full, "GET", "/api/v1/access-grants", "access_token_route_forbidden"},
		{"workspaces mixed case and slash", full, "GET", "/API/V1/Workspaces/", "access_token_route_forbidden"},
		{"workspace members", full, "PUT", "/api/v1/workspaces/afl/members/u", "access_token_route_forbidden"},
		{"ai catalog", full, "GET", "/api/v1/ai/provider-connections", "access_token_route_forbidden"},
		{"import", full, "POST", "/api/v1/domain/import/plan", "access_token_route_forbidden"},
		{"integrations", full, "GET", "/api/v1/integrations", "access_token_route_forbidden"},
		{"delete without flag", write, "DELETE", "/api/v1/queries/q1", "access_token_delete_forbidden"},
		{"delete with flag", full, "DELETE", "/api/v1/queries/q1", ""},
		{"commit without flag", write, "POST", "/api/v1/commits", "access_token_commit_forbidden"},
		{"commit restore without flag", write, "POST", "/api/v1/commits/c1/restore", "access_token_commit_forbidden"},
		{"release without flag", write, "POST", "/api/v1/releases/", "access_token_commit_forbidden"},
		{"commit plan is allowed", write, "POST", "/api/v1/commits/plan", ""},
		{"commit with flag", full, "POST", "/api/v1/commits", ""},
		{"backups list", full, "GET", "/api/v1/domain/backups", "access_token_route_forbidden"},
		{"backup export", full, "GET", "/api/v1/domain/backups/b1/export", "access_token_route_forbidden"},
		{"release bundle", full, "GET", "/api/v1/releases/r1/bundle", "access_token_route_forbidden"},
		{"release export", full, "GET", "/api/v1/releases/r1/export/", "access_token_route_forbidden"},
		{"release read", read, "GET", "/api/v1/releases/r1", ""},
		{"domain live", read, "GET", "/api/v1/domain", ""},
		{"domain export", read, "GET", "/api/v1/domain/export", ""},
		{"base path", read, "GET", "/endge/api/v1/workspaces", "access_token_route_forbidden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiPrefix := "/api"
			if tc.name == "base path" {
				apiPrefix = "/endge/api"
			}
			err := accessTokenRouteError(tc.principal, tc.method, normalizeRoutePath(tc.path, apiPrefix))
			if code := domainerrors.CodeOf(err); (err == nil && tc.code != "") || (err != nil && code != tc.code) {
				t.Fatalf("expected %q, got %v", tc.code, err)
			}
			if err != nil && domainerrors.HTTPStatusOf(err) != 403 {
				t.Fatalf("policy errors must be 403, got %d", domainerrors.HTTPStatusOf(err))
			}
		})
	}
}
