package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// TestOIDCResolverAcceptsKeycloakStyleRSAClaims проверяет совместимый с Keycloak RSA token.
func TestOIDCResolverAcceptsKeycloakStyleRSAClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer server.Close()
	identity := config.IdentityConfig{Mode: "oidc", ProviderID: "keycloak", Issuer: server.URL, JWKSURL: server.URL, AllowedAudiences: []string{"endge-configurator"}, AllowedAlgorithms: []string{"RS256"}, UsernameClaim: "preferred_username", DisplayNameClaim: "name", GroupsClaim: "groups", PlatformAdminGroups: []string{"endge-platform-admins"}}
	resolver, err := NewResolver(&config.Config{Identity: identity})
	if err != nil {
		t.Fatalf("create resolver: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": server.URL, "aud": []string{"endge-configurator"}, "sub": "42", "exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(-time.Minute).Unix(), "preferred_username": "alice", "name": "Alice", "groups": []string{"endge-platform-admins"}})
	token.Header["kid"] = "test-key"
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := resolver.Resolve(context.Background(), raw)
	if err != nil {
		t.Fatalf("resolve token: %v", err)
	}
	if claims.Subject != "42" || claims.Username != "alice" || !claims.PlatformAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

// TestOIDCResolverRejectsWrongAudience проверяет обязательную аудиторию OIDC token.
func TestOIDCResolverRejectsWrongAudience(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer server.Close()
	resolver, err := NewResolver(&config.Config{Identity: config.IdentityConfig{Mode: "oidc", ProviderID: "primary", Issuer: server.URL, JWKSURL: server.URL, AllowedAudiences: []string{"expected"}, AllowedAlgorithms: []string{"RS256"}}})
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": server.URL, "aud": "wrong", "sub": "42", "exp": time.Now().Add(time.Hour).Unix()})
	token.Header["kid"] = "key"
	raw, _ := token.SignedString(key)
	if _, err = resolver.Resolve(context.Background(), raw); err == nil {
		t.Fatal("wrong audience token was accepted")
	}
}

// TestOIDCResolverRejectsUnsafeTokenVariants проверяет обязательные OIDC claims, kid, алгоритм и подпись.
func TestOIDCResolverRejectsUnsafeTokenVariants(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("создать основной RSA key: %v", err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("создать неверный RSA key: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "safe-key", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	resolver, err := NewResolver(&config.Config{Identity: config.IdentityConfig{
		Mode: "oidc", ProviderID: "primary", Issuer: server.URL, JWKSURL: server.URL,
		AllowedAudiences: []string{"endge-configurator"}, AllowedAlgorithms: []string{"RS256"},
	}})
	if err != nil {
		t.Fatalf("создать resolver: %v", err)
	}
	validClaims := func() jwt.MapClaims {
		return jwt.MapClaims{"iss": server.URL, "aud": "endge-configurator", "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(-time.Minute).Unix()}
	}
	tests := []struct {
		name   string
		claims func(jwt.MapClaims)
		key    *rsa.PrivateKey
		kid    string
	}{
		{name: "нет subject", claims: func(value jwt.MapClaims) { delete(value, "sub") }, key: key, kid: "safe-key"},
		{name: "нет expiration", claims: func(value jwt.MapClaims) { delete(value, "exp") }, key: key, kid: "safe-key"},
		{name: "истёкший token", claims: func(value jwt.MapClaims) { value["exp"] = time.Now().Add(-time.Hour).Unix() }, key: key, kid: "safe-key"},
		{name: "будущий nbf", claims: func(value jwt.MapClaims) { value["nbf"] = time.Now().Add(time.Hour).Unix() }, key: key, kid: "safe-key"},
		{name: "неверный issuer", claims: func(value jwt.MapClaims) { value["iss"] = "https://attacker.invalid" }, key: key, kid: "safe-key"},
		{name: "неверный audience", claims: func(value jwt.MapClaims) { value["aud"] = "other" }, key: key, kid: "safe-key"},
		{name: "нет kid", claims: func(jwt.MapClaims) {}, key: key, kid: ""},
		{name: "неизвестный kid", claims: func(jwt.MapClaims) {}, key: key, kid: "unknown-key"},
		{name: "неверная подпись", claims: func(jwt.MapClaims) {}, key: wrongKey, kid: "safe-key"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			claims := validClaims()
			testCase.claims(claims)
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			if testCase.kid != "" {
				token.Header["kid"] = testCase.kid
			}
			raw, signErr := token.SignedString(testCase.key)
			if signErr != nil {
				t.Fatalf("подписать token: %v", signErr)
			}
			if _, resolveErr := resolver.Resolve(context.Background(), raw); resolveErr == nil {
				t.Fatal("небезопасный token был принят")
			}
		})
	}

	hmac := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
	hmac.Header["kid"] = "safe-key"
	rawHMAC, err := hmac.SignedString([]byte("shared-secret"))
	if err != nil {
		t.Fatalf("подписать HMAC token: %v", err)
	}
	if _, err = resolver.Resolve(context.Background(), rawHMAC); err == nil {
		t.Fatal("HMAC token был принят JWKS resolver")
	}
}
