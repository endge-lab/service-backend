package documents

import (
	"encoding/json"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// TestValidateSecretsRejectsCredentialMaterial проверяет запрет сохранения client secret.
func TestValidateSecretsRejectsCredentialMaterial(t *testing.T) {
	input := map[string]any{"config": map[string]any{"clientSecret": "value"}}
	if err := validateSecrets(input); err == nil {
		t.Fatal("clientSecret was accepted")
	}
}

// TestValidateSecretsAllowsCredentialsAndTokenEndpoint проверяет выделенную credential boundary.
func TestValidateSecretsAllowsCredentialsAndTokenEndpoint(t *testing.T) {
	input := map[string]any{"credentials": map[string]any{"clientSecret": "literal-test-secret"}, "tokenEndpoint": "https://issuer/token"}
	if err := validateSecrets(input); err != nil {
		t.Fatalf("public auth config rejected: %v", err)
	}
}

func TestAuthProfileRejectsCredentialMaterialInsideConfig(t *testing.T) {
	input := map[string]any{
		"identity":    "form-auth",
		"displayName": "Form auth",
		"config":      map[string]any{"password": "configured-at-workspace-level"},
	}
	if err := validateDocument("auth-profiles", input); err == nil {
		t.Fatal("AuthProfile config password was accepted")
	}

	input["adapterId"] = "basic"
	input["config"] = map[string]any{}
	input["credentials"] = map[string]any{"username": "test", "password": "{AODB_PASSWORD}"}
	if err := validateDocument("auth-profiles", input); err != nil {
		t.Fatalf("AuthProfile credentials rejected: %v", err)
	}
}

func TestAuthProfileValidatesBuiltinAdapters(t *testing.T) {
	cases := []map[string]any{
		{"identity": "oidc", "displayName": "OIDC", "adapterId": "oidc", "config": map[string]any{"issuer": "{OIDC_ISSUER}", "clientId": "web", "scopes": []any{"openid", "profile"}}, "credentials": map[string]any{}, "session": map[string]any{"storage": "memory", "persistRefreshToken": false}},
		{"identity": "service", "displayName": "Service", "adapterId": "oauth2-client-credentials", "config": map[string]any{"tokenEndpoint": "{SERVICE_TOKEN_ENDPOINT}", "clientId": "{SERVICE_CLIENT_ID}", "scopes": []any{}, "clientAuthentication": "client_secret_basic"}, "credentials": map[string]any{"clientSecret": "{SERVICE_CLIENT_SECRET}"}, "session": map[string]any{"storage": "sessionStorage", "persistRefreshToken": false}},
		{"identity": "basic", "displayName": "Basic", "adapterId": "basic", "config": map[string]any{}, "credentials": map[string]any{"username": "test", "password": "literal-password"}},
		{"identity": "bearer", "displayName": "Bearer", "adapterId": "bearer", "config": map[string]any{}, "credentials": map[string]any{"token": "{SERVICE_TOKEN}"}},
	}
	for _, input := range cases {
		if err := validateDocument("auth-profiles", input); err != nil {
			t.Fatalf("valid %s profile rejected: %v", input["adapterId"], err)
		}
	}
	cases[0]["config"].(map[string]any)["tokenPath"] = "/token"
	if err := validateDocument("auth-profiles", cases[0]); err == nil {
		t.Fatal("legacy manual OIDC endpoint was accepted")
	}
	cases[2]["session"] = map[string]any{"storage": "memory", "persistRefreshToken": false}
	if err := validateDocument("auth-profiles", cases[2]); err == nil {
		t.Fatal("Basic session policy was accepted")
	}
}

func TestAuthProfileRejectsLegacyTopLevelFields(t *testing.T) {
	input := map[string]any{
		"identity": "oidc", "displayName": "OIDC", "adapterId": "oidc",
		"config":      map[string]any{"issuer": "{OIDC_ISSUER}", "clientId": "web", "scopes": []any{"openid", "profile"}},
		"credentials": map[string]any{},
		"session":     map[string]any{"storage": "memory", "persistRefreshToken": false},
		"loginMode":   "interactive",
	}
	if err := validateDocument("auth-profiles", input); err == nil {
		t.Fatal("legacy top-level auth profile field was accepted")
	}
}

func TestAuthProfileRejectsNestedCredentialsBoundary(t *testing.T) {
	input := map[string]any{
		"identity": "plugin", "displayName": "Plugin", "adapterId": "third-party",
		"config":      map[string]any{"credentials": map[string]any{"token": "literal"}},
		"credentials": map[string]any{},
	}
	if err := validateDocument("auth-profiles", input); err == nil {
		t.Fatal("nested credentials boundary was accepted")
	}
}

// TestQueryRequiresSourceVersionTwo проверяет фиксированную версию Query source-контракта.
func TestQueryRequiresSourceVersionTwo(t *testing.T) {
	input := map[string]any{"identity": "q", "displayName": "Q", "source": "query {}", "sourceVersion": float64(1)}
	if err := validateDocument("queries", input); err == nil {
		t.Fatal("Query sourceVersion 1 was accepted")
	}
}

// TestProjectAcceptsCanonicalNavigationRelation проверяет identity-based Project-навигацию.
func TestProjectAcceptsCanonicalNavigationRelation(t *testing.T) {
	input := map[string]any{"identity": "p", "displayName": "Project", "navigationIdentity": "main"}
	if err := validateDocument("projects", input); err != nil {
		t.Fatalf("canonical Project navigation relation rejected: %v", err)
	}
	input["navigationId"] = 42
	if err := validateDocument("projects", input); err == nil {
		t.Fatal("legacy Project navigationId was accepted")
	}
}

// TestSourceWithoutSourceVersionForMockAndComponent проверяет разные version-контракты source-документов.
func TestSourceWithoutSourceVersionForMockAndComponent(t *testing.T) {
	for _, kind := range []string{"mocks", "components"} {
		input := map[string]any{"identity": "example", "displayName": "Example", "source": "source"}
		if err := validateDocument(kind, input); err != nil {
			t.Fatalf("%s не должен требовать sourceVersion: %v", kind, err)
		}
	}
}

// TestChecksumContentIgnoresJSONFormatting проверяет семантическое no-op сравнение JSONB.
func TestChecksumContentIgnoresJSONFormatting(t *testing.T) {
	first := entities.Document{Identity: "example", DisplayName: "Example", ManagedBy: "user", Meta: json.RawMessage(`{"b":2,"a":1}`), Data: json.RawMessage(`{"nested":{"x":1,"y":2}}`), Active: true}
	second := first
	second.Meta = json.RawMessage(`{ "a": 1, "b": 2 }`)
	second.Data = json.RawMessage(`{"nested": {"y": 2, "x": 1}}`)
	if checksumContent(first) != checksumContent(second) {
		t.Fatal("семантически одинаковый JSON имеет разные checksum")
	}
}
