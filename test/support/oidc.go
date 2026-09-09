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
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IdentityProvider — локальный OIDC/JWKS server без внешней сети и credentials.
type IdentityProvider struct {
	key           *rsa.PrivateKey
	server        *httptest.Server
	mu            sync.Mutex
	codes         map[string]identityTokens
	refreshTokens identityTokens
	refreshCalls  int
	logoutCalls   int
}

type identityTokens struct {
	Access   string
	Identity string
}

// TokenInput задаёт identity и роль пользователя в E2E-сценарии.
type TokenInput struct {
	Subject     string
	Username    string
	DisplayName string
	Groups      []string
	ExpiresAt   time.Time
	IssuedAt    time.Time
	Claims      map[string]any
}

// NewIdentityProvider создаёт RSA key и публикует только public JWKS.
func NewIdentityProvider(t testing.TB) *IdentityProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("создать тестовый RSA key: %v", err)
	}
	provider := &IdentityProvider{key: key, codes: make(map[string]identityTokens)}
	provider.server = httptest.NewServer(http.HandlerFunc(provider.serveHTTP))
	t.Cleanup(provider.server.Close)
	return provider
}

// URL возвращает issuer локального провайдера.
func (p *IdentityProvider) URL() string { return p.server.URL }

// Token подписывает OIDC bearer token для тестового пользователя.
func (p *IdentityProvider) Token(t testing.TB, input TokenInput) string {
	return p.token(t, input, "")
}

// AuthorizationCode creates a one-time OIDC code bound to the supplied nonce.
// The test browser passes it to /auth/callback; /token consumes it exactly once.
func (p *IdentityProvider) AuthorizationCode(t testing.TB, input TokenInput, nonce string) string {
	t.Helper()
	code := randomValue(t, 24)
	p.mu.Lock()
	token := p.token(t, input, nonce)
	p.codes[code] = identityTokens{Access: token, Identity: token}
	p.mu.Unlock()
	return code
}

// AuthorizationCodeWithTokens tests identity/access token separation in the callback.
func (p *IdentityProvider) AuthorizationCodeWithTokens(t testing.TB, identity, access TokenInput, nonce string) string {
	t.Helper()
	code := randomValue(t, 24)
	identityToken, accessToken := p.token(t, identity, nonce), p.token(t, access, "")
	p.mu.Lock()
	defer p.mu.Unlock()
	p.codes[code] = identityTokens{Access: accessToken, Identity: identityToken}
	return code
}

// SetRefreshTokens supplies the next refresh response; an empty value rejects refresh.
func (p *IdentityProvider) SetRefreshTokens(t testing.TB, identity, access TokenInput) {
	t.Helper()
	identityToken, accessToken := p.token(t, identity, ""), p.token(t, access, "")
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refreshTokens = identityTokens{Access: accessToken, Identity: identityToken}
}
func (p *IdentityProvider) RefreshCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refreshCalls
}

// LogoutCalls returns the number of provider logout requests received.
func (p *IdentityProvider) LogoutCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.logoutCalls
}

func (p *IdentityProvider) token(t testing.TB, input TokenInput, nonce string) string {
	t.Helper()
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().Add(time.Hour)
	}
	if input.IssuedAt.IsZero() {
		input.IssuedAt = time.Now()
	}
	claims := jwt.MapClaims{
		"iss": p.URL(), "aud": []string{"endge-configurator"}, "sub": input.Subject,
		"iat": input.IssuedAt.Unix(), "exp": input.ExpiresAt.Unix(), "nbf": time.Now().Add(-time.Minute).Unix(),
		"preferred_username": input.Username, "name": input.DisplayName, "groups": input.Groups, "nonce": nonce,
	}
	for key, value := range input.Claims {
		claims[key] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "endge-test-key"
	raw, err := token.SignedString(p.key)
	if err != nil {
		t.Fatalf("подписать тестовый JWT: %v", err)
	}
	return raw
}

func randomValue(t testing.TB, size int) string {
	t.Helper()
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("создать тестовое OIDC значение: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
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
	case "/token":
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		code := request.FormValue("code")
		p.mu.Lock()
		token, exists := p.codes[code]
		delete(p.codes, code)
		if request.FormValue("grant_type") == "refresh_token" {
			p.refreshCalls++
			token = p.refreshTokens
			exists = token.Access != "" && request.FormValue("refresh_token") == "test-refresh-token"
		}
		p.mu.Unlock()
		if !exists {
			http.Error(response, "invalid authorization code", http.StatusUnauthorized)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"access_token": token.Access, "id_token": token.Identity, "refresh_token": "test-refresh-token", "expires_in": 3600,
		})
	case "/logout":
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.mu.Lock()
		p.logoutCalls++
		p.mu.Unlock()
		response.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(response, request)
	}
}
