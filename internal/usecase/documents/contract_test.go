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

// TestValidateSecretsAllowsCredentialRefsAndTokenEndpoint проверяет допустимые публичные ссылки и URL.
func TestValidateSecretsAllowsCredentialRefsAndTokenEndpoint(t *testing.T) {
	input := map[string]any{"credentialRefs": []any{"oidc-client"}, "tokenEndpoint": "https://issuer/token"}
	if err := validateSecrets(input); err != nil {
		t.Fatalf("public auth config rejected: %v", err)
	}
}

func TestAuthProfileAllowsAdapterCredentialConfig(t *testing.T) {
	input := map[string]any{
		"identity":    "form-auth",
		"displayName": "Form auth",
		"config":      map[string]any{"password": "configured-at-workspace-level"},
	}
	if err := validateDocument("auth-profiles", input); err != nil {
		t.Fatalf("AuthProfile adapter config rejected: %v", err)
	}

	input["clientSecret"] = "outside-adapter-config"
	if err := validateDocument("auth-profiles", input); err == nil {
		t.Fatal("top-level AuthProfile secret was accepted")
	}
}

// TestQueryRequiresSourceVersionTwo проверяет фиксированную версию Query source-контракта.
func TestQueryRequiresSourceVersionTwo(t *testing.T) {
	input := map[string]any{"identity": "q", "displayName": "Q", "source": "query {}", "sourceVersion": float64(1)}
	if err := validateDocument("queries", input); err == nil {
		t.Fatal("Query sourceVersion 1 was accepted")
	}
}

// TestProjectRejectsNavigationRelation проверяет удалённую из MVP Project-навигацию.
func TestProjectRejectsNavigationRelation(t *testing.T) {
	input := map[string]any{"identity": "p", "displayName": "Project", "navigationIdentity": "main"}
	if err := validateDocument("projects", input); err == nil {
		t.Fatal("Project navigation relation was accepted")
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
