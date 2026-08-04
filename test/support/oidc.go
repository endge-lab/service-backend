//go:build e2e

package support

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IdentityProvider — локальный OIDC/JWKS server без внешней сети и credentials.
type IdentityProvider struct {
	key    *rsa.PrivateKey
	server *httptest.Server
}

// TokenInput задаёт identity и роль пользователя в E2E-сценарии.
type TokenInput struct {
	Subject     string
	Username    string
	DisplayName string
	Groups      []string
	ExpiresAt   time.Time
}

// NewIdentityProvider создаёт RSA key и публикует только public JWKS.
func NewIdentityProvider(t testing.TB) *IdentityProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("создать тестовый RSA key: %v", err)
	}
	provider := &IdentityProvider{key: key}
	provider.server = httptest.NewServer(http.HandlerFunc(provider.serveHTTP))
	t.Cleanup(provider.server.Close)
	return provider
}

// URL возвращает issuer локального провайдера.
func (p *IdentityProvider) URL() string { return p.server.URL }

// Token подписывает OIDC bearer token для тестового пользователя.
func (p *IdentityProvider) Token(t testing.TB, input TokenInput) string {
	t.Helper()
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().Add(time.Hour)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": p.URL(), "aud": []string{"endge-configurator"}, "sub": input.Subject,
		"exp": input.ExpiresAt.Unix(), "nbf": time.Now().Add(-time.Minute).Unix(),
		"preferred_username": input.Username, "name": input.DisplayName, "groups": input.Groups,
	})
	token.Header["kid"] = "endge-test-key"
	raw, err := token.SignedString(p.key)
	if err != nil {
		t.Fatalf("подписать тестовый JWT: %v", err)
	}
	return raw
}

func (p *IdentityProvider) serveHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/jwks":
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "endge-test-key", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(p.key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(p.key.E)).Bytes()),
		}}})
	default:
		http.NotFound(response, request)
	}
}
