package config

import (
	"encoding/base64"
	"testing"
	"time"
)

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

// TestLoadFromEnvironmentWithoutYAML проверяет container-first конфигурацию:
// все средовые значения могут прийти через process environment без config file.
func TestLoadFromEnvironmentWithoutYAML(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_NAME", "endge-service-backend")
	t.Setenv("PUBLIC_URL", "https://backend.example.test")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://configurator.example.test")
	t.Setenv("POSTGRES_HOST", "postgres.example.test")
	t.Setenv("POSTGRES_DATABASE", "endge_service_backend")
	t.Setenv("AUTH_MODE", "dev")
	t.Setenv("AUTH_LOGIN_ADAPTER", "dev")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Name != "endge-service-backend" {
		t.Fatalf("app name=%q, want environment value", cfg.App.Name)
	}
	if cfg.Postgres.Host != "postgres.example.test" || cfg.Postgres.Database != "endge_service_backend" {
		t.Fatalf("postgres config mismatch: %#v", cfg.Postgres)
	}
}

// TestConfiguratorAuthConfigAcceptsEncryptionKeyRotation проверяет валидную
// конфигурацию текущего и предыдущего ключей server-side sessions.
func TestConfiguratorAuthConfigAcceptsEncryptionKeyRotation(t *testing.T) {
	value := validConfiguratorAuthConfig()
	value.SessionEncryptionKeyID = "v2"
	value.SessionPreviousEncryptionKeys = []SessionEncryptionKeyConfig{{ID: "v1", Key: testEncryptionKey(2)}}
	if err := value.Validate(true); err != nil {
		t.Fatalf("валидная ротация ключа отклонена: %v", err)
	}
}

// TestConfiguratorAuthConfigRejectsDuplicateEncryptionKeyID проверяет, что
// один key id нельзя одновременно назначить текущему и предыдущему ключу.
func TestConfiguratorAuthConfigRejectsDuplicateEncryptionKeyID(t *testing.T) {
	value := validConfiguratorAuthConfig()
	value.SessionPreviousEncryptionKeys = []SessionEncryptionKeyConfig{{ID: value.SessionEncryptionKeyID, Key: testEncryptionKey(2)}}
	if err := value.Validate(true); err == nil {
		t.Fatal("дублирующийся encryption key id был принят")
	}
}

func TestReleaseArtifactCacheConfigRejectsInvalidEnabledLimits(t *testing.T) {
	value := ReleaseArtifactCacheConfig{Enabled: true, MaxBytes: 0, MaxItemBytes: 1}
	if err := value.Validate(); err == nil {
		t.Fatal("enabled cache accepted zero total limit")
	}
	value = ReleaseArtifactCacheConfig{Enabled: true, MaxBytes: 1, MaxItemBytes: 0}
	if err := value.Validate(); err == nil {
		t.Fatal("enabled cache accepted zero item limit")
	}
	if err := (ReleaseArtifactCacheConfig{Enabled: false}).Validate(); err != nil {
		t.Fatalf("disabled cache rejected: %v", err)
	}
}

func validConfiguratorAuthConfig() ConfiguratorAuthConfig {
	return ConfiguratorAuthConfig{
		Adapter: "oidc", AuthorizationURL: "https://issuer.example/authorize", TokenURL: "https://issuer.example/token",
		ClientID: "configurator", RedirectURL: "https://backend.example/auth/callback", ReturnURL: "https://configurator.example",
		SessionEncryptionKeyID: "v1", SessionEncryptionKey: testEncryptionKey(1), SessionTTL: time.Hour,
		TransactionTTL: time.Minute, SessionCleanupInterval: time.Minute, CookieSecure: true,
	}
}

func testEncryptionKey(fill byte) string {
	value := make([]byte, 32)
	for index := range value {
		value[index] = fill
	}
	return base64.StdEncoding.EncodeToString(value)
}
