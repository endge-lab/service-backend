package config

import "testing"

// TestIdentityConfigRejectsUnsafeProductionModes проверяет fail-closed production auth.
func TestIdentityConfigRejectsUnsafeProductionModes(t *testing.T) {
	if err := (IdentityConfig{Mode: "dev", DevSubject: "developer"}).Validate(true); err == nil {
		t.Fatal("production accepted dev auth")
	}
	value := IdentityConfig{Mode: "oidc", ProviderID: "primary", Issuer: "https://issuer.example", JWKSURL: "https://issuer.example/jwks", AllowedAudiences: []string{"endge"}, AllowedAlgorithms: []string{"HS256"}}
	if err := value.Validate(true); err == nil {
		t.Fatal("OIDC accepted HMAC algorithm")
	}
}

// TestIdentityConfigAcceptsRSAOIDC проверяет корректную production OIDC-конфигурацию.
func TestIdentityConfigAcceptsRSAOIDC(t *testing.T) {
	value := IdentityConfig{Mode: "oidc", ProviderID: "primary", Issuer: "https://issuer.example", JWKSURL: "https://issuer.example/jwks", AllowedAudiences: []string{"endge"}, AllowedAlgorithms: []string{"RS256"}}
	if err := value.Validate(true); err != nil {
		t.Fatalf("valid OIDC config rejected: %v", err)
	}
}

// TestLoadHonorsPostgresDatabaseEnvironment проверяет явный override имени runtime-БД.
func TestLoadHonorsPostgresDatabaseEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "codex")
	t.Setenv("CONFIG_PATH", "../../development.yaml")
	t.Setenv("POSTGRES_DATABASE", "endge_config_test")
	t.Setenv("AUTH_MODE", "dev")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Postgres.Database != "endge_config_test" {
		t.Fatalf("database=%q, want environment value", cfg.Postgres.Database)
	}
}
