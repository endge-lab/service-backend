package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/access_tokens"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

// tokenRepo находит учётную запись по SHA-256 секрета; остальные методы не нужны middleware.
type tokenRepo struct {
	ports.AccessTokenRepository
	credentials map[string]ports.AccessTokenCredential
}

func (r tokenRepo) FindAccessTokenCredential(_ context.Context, hash []byte) (*ports.AccessTokenCredential, error) {
	credential, ok := r.credentials[string(hash)]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return &credential, nil
}

func (tokenRepo) TouchAccessToken(context.Context, string, time.Duration) error { return nil }

func testSecret(fill byte) string {
	return entities.AccessTokenSecretPrefix + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat(string(fill), 32)))
}

func credentialFor(secret string, token entities.AccessToken) (string, ports.AccessTokenCredential) {
	sum := sha256.Sum256([]byte(secret))
	return string(sum[:]), ports.AccessTokenCredential{Token: token, SecretHash: sum[:], Owner: entities.User{ID: "u-1", Username: "egor", DisplayName: "Егор"}}
}

func newTokenApp(t *testing.T) (*fiber.App, map[string]string) {
	t.Helper()
	now := time.Now()
	revokedAt := now.Add(-time.Minute)
	secrets := map[string]string{"read": testSecret('r'), "write": testSecret('w'), "revoked": testSecret('x'), "expired": testSecret('e'), "unknown": testSecret('u')}
	repo := tokenRepo{credentials: map[string]ports.AccessTokenCredential{}}
	for name, token := range map[string]entities.AccessToken{
		"read":    {ID: "t-read", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeRead, ExpiresAt: now.Add(time.Hour)},
		"write":   {ID: "t-write", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeWrite, ExpiresAt: now.Add(time.Hour)},
		"revoked": {ID: "t-revoked", Scope: entities.AccessTokenScopeRead, ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt},
		"expired": {ID: "t-expired", Scope: entities.AccessTokenScopeRead, ExpiresAt: now.Add(-time.Hour)},
	} {
		key, credential := credentialFor(secrets[name], token)
		repo.credentials[key] = credential
	}
	m := &authMiddleware{tokens: access_tokens.NewUseCase(repo, nil, 720*time.Hour)}
	app := fiber.New()
	app.Use(m.AuthMiddleware(), AccessTokenPolicy("/endge"))
	app.All("/endge/api/*", func(c *fiber.Ctx) error {
		token, _ := entities.AccessTokenFromContext(c.UserContext())
		identity, _ := IdentityFromContext(c.UserContext())
		return c.JSON(fiber.Map{"token": token.TokenID, "scope": token.Scope, "user": identity.AuthUserID, "accessTokenId": identity.AccessTokenID})
	})
	return app, secrets
}

func doTokenRequest(t *testing.T, app *fiber.App, method, target, secret string) (int, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+secret)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	raw, _ := io.ReadAll(response.Body)
	if strings.Contains(string(raw), secret) {
		t.Fatalf("response must not echo the secret: %s", raw)
	}
	body := map[string]any{}
	_ = json.Unmarshal(raw, &body)
	return response.StatusCode, body
}

// TestAuthMiddlewareAuthenticatesAccessToken проверяет PAT до JWT: identity владельца и
// ограничения токена в context, 401 с кодом для неизвестного, отозванного и истёкшего токена.
func TestAuthMiddlewareAuthenticatesAccessToken(t *testing.T) {
	app, secrets := newTokenApp(t)
	status, body := doTokenRequest(t, app, fiber.MethodGet, "/endge/api/v1/queries", secrets["read"])
	if status != fiber.StatusOK || body["token"] != "t-read" || body["scope"] != "read" || body["user"] != "u-1" || body["accessTokenId"] != "t-read" {
		t.Fatalf("read token: %d %v", status, body)
	}
	for name, code := range map[string]string{"unknown": "access_token_invalid", "revoked": "access_token_revoked", "expired": "access_token_expired"} {
		status, body = doTokenRequest(t, app, fiber.MethodGet, "/endge/api/v1/queries", secrets[name])
		if status != fiber.StatusUnauthorized || body["code"] != code {
			t.Fatalf("%s token: %d %v", name, status, body)
		}
	}
	// Секрет с префиксом, но неверного формата — тоже 401, без обращения к JWT.
	status, body = doTokenRequest(t, app, fiber.MethodGet, "/endge/api/v1/queries", entities.AccessTokenSecretPrefix+"short")
	if status != fiber.StatusUnauthorized || body["code"] != "access_token_invalid" {
		t.Fatalf("malformed token: %d %v", status, body)
	}
}

// TestAccessTokenPolicyHandlerAppliesFlags проверяет handler политики на реальном пути с HTTP_BASE_PATH.
func TestAccessTokenPolicyHandlerAppliesFlags(t *testing.T) {
	app, secrets := newTokenApp(t)
	cases := []struct {
		secret, method, path string
		status               int
		code                 string
	}{
		{secrets["write"], fiber.MethodGet, "/endge/api/v1/queries", fiber.StatusOK, ""},
		{secrets["write"], fiber.MethodDelete, "/endge/api/v1/queries/q", fiber.StatusForbidden, "access_token_delete_forbidden"},
		{secrets["write"], fiber.MethodPost, "/endge/api/v1/commits", fiber.StatusForbidden, "access_token_commit_forbidden"},
		{secrets["write"], fiber.MethodPost, "/endge/api/v1/commits/plan", fiber.StatusOK, ""},
		{secrets["write"], fiber.MethodGet, "/endge/API/V1/Access-Tokens/", fiber.StatusForbidden, "access_token_management_forbidden"},
		{secrets["read"], fiber.MethodGet, "/endge/api/v1/domain/backups", fiber.StatusForbidden, "access_token_route_forbidden"},
	}
	for _, tc := range cases {
		status, body := doTokenRequest(t, app, tc.method, tc.path, tc.secret)
		if status != tc.status || (tc.code != "" && body["code"] != tc.code) {
			t.Fatalf("%s %s: %d %v, want %d %s", tc.method, tc.path, status, body, tc.status, tc.code)
		}
	}
}
