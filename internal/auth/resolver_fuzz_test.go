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

	"github.com/endge-lab/service-backend/internal/config"
)

// FuzzOIDCResolverMalformedToken проверяет, что произвольный bearer token не вызывает panic и не раскрывает parser internals.
func FuzzOIDCResolverMalformedToken(f *testing.F) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		f.Fatalf("создать RSA key: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(response).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "fuzz-key", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	resolver, err := NewResolver(&config.Config{Identity: config.IdentityConfig{
		Mode: "oidc", ProviderID: "fuzz", Issuer: server.URL, JWKSURL: server.URL,
		AllowedAudiences: []string{"endge-configurator"}, AllowedAlgorithms: []string{"RS256"},
	}})
	if err != nil {
		f.Fatalf("создать OIDC resolver: %v", err)
	}
	for _, seed := range []string{"", ".", "a.b.c", "null", "Bearer token", "eyJhbGciOiJub25lIn0.e30."} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 64*1024 {
			t.Skip()
		}
		_, _ = resolver.Resolve(context.Background(), raw)
	})
}
