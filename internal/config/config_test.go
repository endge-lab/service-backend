package config

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/buildinfo"
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
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey(1))
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
	previousVersion := buildinfo.Version
	previousSchemaVersion := buildinfo.WorkspaceSchemaVersion
	buildinfo.Version = "0.10.0"
	buildinfo.WorkspaceSchemaVersion = "1"
	t.Cleanup(func() {
		buildinfo.Version = previousVersion
		buildinfo.WorkspaceSchemaVersion = previousSchemaVersion
	})
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
	t.Setenv("AUTH_ALLOWED_RETURN_ORIGINS", "https://configurator.example.test,http://localhost:5173")
	t.Setenv("ENCRYPTION_KEY", testEncryptionKey(1))

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
	if len(cfg.ConfiguratorAuth.AllowedReturnOrigins) != 2 || cfg.ConfiguratorAuth.AllowedReturnOrigins[1] != "http://localhost:5173" {
		t.Fatalf("return origins mismatch: %#v", cfg.ConfiguratorAuth.AllowedReturnOrigins)
	}
}

func TestNormalizeHTTPBasePath(t *testing.T) {
	for input, want := range map[string]string{
		"":                       "",
		"/":                      "",
		"/endge-service-backend": "/endge-service-backend",
	} {
		got, err := normalizeHTTPBasePath(input)
		if err != nil || got != want {
			t.Fatalf("normalizeHTTPBasePath(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"endge-service-backend", "/endge-service-backend/", "/endge//backend", "/endge/../backend"} {
		if _, err := normalizeHTTPBasePath(input); err == nil {
			t.Fatalf("normalizeHTTPBasePath(%q) accepted invalid path", input)
		}
	}
}

func TestPublicURLPathMatchesHTTPBasePath(t *testing.T) {
	if err := validatePublicURLBasePath("https://backend.example/endge-service-backend", "/endge-service-backend"); err != nil {
		t.Fatalf("matching paths rejected: %v", err)
	}
	if err := validatePublicURLBasePath("https://backend.example", "/endge-service-backend"); err == nil {
		t.Fatal("PUBLIC_URL without configured HTTP_BASE_PATH was accepted")
	}
}

// TestEncryptionConfigAcceptsKeyRotation проверяет валидную конфигурацию
// общего keyring для server-side sessions и AI credentials.
func TestEncryptionConfigAcceptsKeyRotation(t *testing.T) {
	value := EncryptionConfig{KeyID: "v2", Key: testEncryptionKey(1), PreviousKeys: []EncryptionKeyConfig{{ID: "v1", Key: testEncryptionKey(2)}}}
	if err := value.Validate(); err != nil {
		t.Fatalf("валидная ротация ключа отклонена: %v", err)
	}
}

// TestEncryptionConfigRejectsDuplicateKeyID проверяет, что
// один key id нельзя одновременно назначить текущему и предыдущему ключу.
func TestEncryptionConfigRejectsDuplicateKeyID(t *testing.T) {
	value := EncryptionConfig{KeyID: "v1", Key: testEncryptionKey(1), PreviousKeys: []EncryptionKeyConfig{{ID: "v1", Key: testEncryptionKey(2)}}}
	if err := value.Validate(); err == nil {
		t.Fatal("дублирующийся encryption key id был принят")
	}
}

func TestConfiguratorAuthConfigValidatesCookieSameSite(t *testing.T) {
	value := validConfiguratorAuthConfig()
	value.CookieSameSite = "none"
	if err := value.Validate(true); err != nil {
		t.Fatalf("secure SameSite=None rejected: %v", err)
	}
	value.CookieSecure = false
	if err := value.Validate(false); err == nil {
		t.Fatal("SameSite=None without Secure was accepted")
	}
	value.CookieSameSite = "invalid"
	if err := value.Validate(false); err == nil {
		t.Fatal("invalid SameSite mode was accepted")
	}
}

func TestConfiguratorAuthConfigValidatesAllowedReturnOrigins(t *testing.T) {
	value := validConfiguratorAuthConfig()
	value.AllowedReturnOrigins = []string{"https://configurator.example", "https://configurator-local.example:5173"}
	if err := value.Validate(true); err != nil {
		t.Fatalf("valid return origins rejected: %v", err)
	}
	for _, invalid := range []string{"*", "https://configurator.example/path", "https://user@configurator.example"} {
		value.AllowedReturnOrigins = []string{invalid}
		if err := value.Validate(false); err == nil {
			t.Fatalf("invalid return origin %q was accepted", invalid)
		}
	}
	value.AllowedReturnOrigins = []string{"http://localhost:5173"}
	if err := value.Validate(true); err == nil {
		t.Fatal("production accepted insecure return origin")
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
		SessionTTL: time.Hour, TransactionTTL: time.Minute, SessionCleanupInterval: time.Minute,
		CookieSecure: true, CookieSameSite: "lax",
	}
}

func testEncryptionKey(fill byte) string {
	value := make([]byte, 32)
	for index := range value {
		value[index] = fill
	}
	return base64.StdEncoding.EncodeToString(value)
}
