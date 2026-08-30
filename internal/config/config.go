package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/buildinfo"
	kitconfig "github.com/endge-lab/service-kit-go/config"
)

type BaseConfig = kitconfig.ServiceConfig

type AppConfig = kitconfig.ServiceAppConfig
type HTTPConfig = kitconfig.ServiceHTTPConfig
type LoggerConfig = kitconfig.ServiceLoggerConfig
type MetricsConfig = kitconfig.ServiceMetricsConfig
type PostgresConfig = kitconfig.ServicePostgresConfig
type AuthConfig = kitconfig.ServiceAuthConfig
type TelemetryConfig = kitconfig.ServiceTelemetryConfig
type RedpandaConfig = kitconfig.ServiceRedpandaConfig
type TLSConfig = kitconfig.ServiceTLSConfig

// IdentityConfig belongs to this service because login policy is part of the
// Configurator backend, not a reusable service-kit concern.
type IdentityConfig struct {
	Mode                  string
	ProviderID            string
	Issuer                string
	JWKSURL               string
	AllowedAudiences      []string
	AllowedAlgorithms     []string
	UsernameClaim         string
	DisplayNameClaim      string
	GroupsClaim           string
	PlatformAdminSubjects []string
	PlatformAdminGroups   []string
	DevSubject            string
	DevUsername           string
	DevDisplayName        string
	DevPlatformAdmin      bool
}

// ConfiguratorAuthConfig owns the browser login flow and the server-side
// Configurator session. Provider tokens never leave the backend.
type ConfiguratorAuthConfig struct {
	Adapter                string
	AuthorizationURL       string
	TokenURL               string
	LogoutURL              string
	ClientID               string
	ClientSecret           string
	Scopes                 []string
	RedirectURL            string
	ReturnURL              string
	AllowedReturnOrigins   []string
	SessionCookieName      string
	SessionTTL             time.Duration
	TransactionTTL         time.Duration
	SessionCleanupInterval time.Duration
	CookieSecure           bool
	CookieSameSite         string
	CookieDomain           string
}

type EncryptionKeyConfig struct {
	ID  string
	Key string
}

// EncryptionConfig is shared by browser sessions and encrypted provider credentials.
type EncryptionConfig struct {
	KeyID        string
	Key          string
	PreviousKeys []EncryptionKeyConfig
}

type Config struct {
	*kitconfig.ServiceConfig
	WorkspaceSchemaVersion int
	HTTPBasePath           string
	Identity               IdentityConfig
	ConfiguratorAuth       ConfiguratorAuthConfig
	Encryption             EncryptionConfig
	Snapshots              SnapshotConfig
	ReleaseArtifactCache   ReleaseArtifactCacheConfig
	AIWorkbench            AIWorkbenchConfig
}

// SnapshotConfig задаёт срок хранения временных страховочных копий импорта.
type SnapshotConfig struct {
	ImportBackupRetentionDays int
}

// ReleaseArtifactCacheConfig ограничивает локальный кеш immutable JSON релизов.
// Каждая реплика приложения использует собственный кеш.
type ReleaseArtifactCacheConfig struct {
	Enabled      bool
	MaxBytes     int
	MaxItemBytes int
}

type AIWorkbenchConfig struct {
	GRPCTarget     string
	RequestTimeout time.Duration
	HealthTimeout  time.Duration
	HealthCacheTTL time.Duration
	TLS            kitconfig.ServiceTLSConfig
}

func (c ReleaseArtifactCacheConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.MaxBytes <= 0 {
		return fmt.Errorf("RELEASE_ARTIFACT_CACHE_MAX_BYTES must be positive when cache is enabled")
	}
	if c.MaxItemBytes <= 0 {
		return fmt.Errorf("RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES must be positive when cache is enabled")
	}
	return nil
}

func Load() (*Config, error) {
	base, err := kitconfig.LoadServiceConfig()
	if err != nil {
		return nil, err
	}
	buildMetadata, err := buildinfo.Resolve(base.App.Version)
	if err != nil {
		return nil, fmt.Errorf("load build metadata: %w", err)
	}
	base.App.Version = buildMetadata.AppVersion
	httpBasePath, err := normalizeHTTPBasePath(os.Getenv("HTTP_BASE_PATH"))
	if err != nil {
		return nil, err
	}
	if err := validatePublicURLBasePath(base.App.PublicURL, httpBasePath); err != nil {
		return nil, err
	}

	identity := IdentityConfig{
		Mode:                  env("AUTH_MODE", modeDefault(base.App.IsProduction())),
		ProviderID:            env("AUTH_PROVIDER_ID", "primary"),
		Issuer:                env("AUTH_ISSUER", base.Auth.Issuer),
		JWKSURL:               env("AUTH_JWKS_URL", legacyJWKSURL(base.Auth)),
		AllowedAudiences:      csv(env("AUTH_ALLOWED_AUDIENCES", base.Auth.AllowedAudiences)),
		AllowedAlgorithms:     csv(env("AUTH_ALLOWED_ALGORITHMS", "RS256")),
		UsernameClaim:         env("AUTH_USERNAME_CLAIM", "preferred_username"),
		DisplayNameClaim:      env("AUTH_DISPLAY_NAME_CLAIM", "name"),
		GroupsClaim:           env("AUTH_GROUPS_CLAIM", "groups"),
		PlatformAdminSubjects: csv(os.Getenv("AUTH_PLATFORM_ADMIN_SUBJECTS")),
		PlatformAdminGroups:   csv(os.Getenv("AUTH_PLATFORM_ADMIN_GROUPS")),
		DevSubject:            env("AUTH_DEV_SUBJECT", "developer"),
		DevUsername:           env("AUTH_DEV_USERNAME", "developer"),
		DevDisplayName:        env("AUTH_DEV_DISPLAY_NAME", "Endge Developer"),
		DevPlatformAdmin:      envBool("AUTH_DEV_PLATFORM_ADMIN", true),
	}
	if err := identity.Validate(base.App.IsProduction()); err != nil {
		return nil, err
	}
	previousEncryptionKeys, err := parseEncryptionKeys(os.Getenv("ENCRYPTION_PREVIOUS_KEYS"))
	if err != nil {
		return nil, err
	}
	encryption := EncryptionConfig{
		KeyID: env("ENCRYPTION_KEY_ID", "v1"), Key: strings.TrimSpace(os.Getenv("ENCRYPTION_KEY")), PreviousKeys: previousEncryptionKeys,
	}
	if err := encryption.Validate(); err != nil {
		return nil, err
	}
	configuratorAuth := ConfiguratorAuthConfig{
		Adapter:                env("AUTH_LOGIN_ADAPTER", loginAdapterDefault(identity.Mode)),
		AuthorizationURL:       env("AUTH_AUTHORIZATION_URL", ""),
		TokenURL:               env("AUTH_TOKEN_URL", ""),
		LogoutURL:              env("AUTH_LOGOUT_URL", ""),
		ClientID:               env("AUTH_CLIENT_ID", ""),
		ClientSecret:           strings.TrimSpace(os.Getenv("AUTH_CLIENT_SECRET")),
		Scopes:                 csv(env("AUTH_SCOPES", "openid,profile,email")),
		RedirectURL:            env("AUTH_REDIRECT_URL", strings.TrimRight(base.App.PublicURL, "/")+"/auth/callback"),
		ReturnURL:              env("AUTH_RETURN_URL", strings.TrimRight(base.App.PublicURL, "/")),
		AllowedReturnOrigins:   csv(os.Getenv("AUTH_ALLOWED_RETURN_ORIGINS")),
		SessionCookieName:      env("AUTH_SESSION_COOKIE_NAME", "endge_configurator_session"),
		SessionTTL:             envDuration("AUTH_SESSION_TTL", 8*time.Hour),
		TransactionTTL:         envDuration("AUTH_TRANSACTION_TTL", 10*time.Minute),
		SessionCleanupInterval: envDuration("AUTH_SESSION_CLEANUP_INTERVAL", 15*time.Minute),
		CookieSecure:           envBool("AUTH_COOKIE_SECURE", base.App.IsProduction()),
		CookieSameSite:         strings.ToLower(env("AUTH_COOKIE_SAME_SITE", "lax")),
		CookieDomain:           strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
	}
	if err := configuratorAuth.Validate(base.App.IsProduction()); err != nil {
		return nil, err
	}
	releaseArtifactCache := ReleaseArtifactCacheConfig{
		Enabled:      envBool("RELEASE_ARTIFACT_CACHE_ENABLED", true),
		MaxBytes:     envIntAllowZero("RELEASE_ARTIFACT_CACHE_MAX_BYTES", 64*1024*1024),
		MaxItemBytes: envIntAllowZero("RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES", 16*1024*1024),
	}
	aiWorkbench := AIWorkbenchConfig{
		GRPCTarget:     strings.TrimSpace(os.Getenv("AI_WORKBENCH_GRPC_TARGET")),
		RequestTimeout: envDuration("AI_WORKBENCH_REQUEST_TIMEOUT", 2*time.Minute),
		HealthTimeout:  envDuration("AI_WORKBENCH_HEALTH_TIMEOUT", 2*time.Second),
		HealthCacheTTL: envDuration("AI_WORKBENCH_HEALTH_CACHE_TTL", 5*time.Second),
		TLS: kitconfig.ServiceTLSConfig{
			Enabled: envBool("AI_WORKBENCH_TLS_ENABLED", false), CertFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_CERT_FILE")),
			KeyFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_KEY_FILE")), CAFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_CA_FILE")),
			InsecureSkipVerify: envBool("AI_WORKBENCH_TLS_INSECURE_SKIP_VERIFY", false),
		},
	}
	if aiWorkbench.RequestTimeout <= 0 || aiWorkbench.HealthTimeout <= 0 || aiWorkbench.HealthCacheTTL <= 0 {
		return nil, fmt.Errorf("AI Workbench timeouts must be positive")
	}
	if aiWorkbench.GRPCTarget != "" && base.App.IsProduction() && !base.Identity.Client.Enabled {
		return nil, fmt.Errorf("service identity client must be enabled when AI Workbench is configured in production")
	}
	if err := releaseArtifactCache.Validate(); err != nil {
		return nil, err
	}
	return &Config{
		ServiceConfig:          base,
		WorkspaceSchemaVersion: buildMetadata.WorkspaceSchemaVersion,
		HTTPBasePath:           httpBasePath,
		Identity:               identity,
		ConfiguratorAuth:       configuratorAuth,
		Encryption:             encryption,
		ReleaseArtifactCache:   releaseArtifactCache,
		AIWorkbench:            aiWorkbench,
		Snapshots: SnapshotConfig{
			ImportBackupRetentionDays: envInt("IMPORT_BACKUP_RETENTION_DAYS", 7),
		},
	}, nil
}

func normalizeHTTPBasePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == "/" {
		return "", nil
	}
	if !strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("HTTP_BASE_PATH must start with /")
	}
	if strings.ContainsAny(value, "?#\\%") || path.Clean(value) != value {
		return "", fmt.Errorf("HTTP_BASE_PATH must be a clean URL path without a trailing slash")
	}
	return value, nil
}

func validatePublicURLBasePath(publicURL, basePath string) error {
	if basePath == "" {
		return nil
	}
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("PUBLIC_URL must be a valid URL: %w", err)
	}
	if strings.TrimRight(parsed.Path, "/") != basePath {
		return fmt.Errorf("PUBLIC_URL path must match HTTP_BASE_PATH")
	}
	return nil
}

func (c ConfiguratorAuthConfig) Validate(production bool) error {
	if c.CookieSameSite != "lax" && c.CookieSameSite != "strict" && c.CookieSameSite != "none" {
		return fmt.Errorf("AUTH_COOKIE_SAME_SITE must be lax, strict or none")
	}
	if c.CookieSameSite == "none" && !c.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true when AUTH_COOKIE_SAME_SITE=none")
	}
	if err := validateAuthHTTPURL("AUTH_RETURN_URL", c.ReturnURL, production, false); err != nil {
		return err
	}
	for _, origin := range c.AllowedReturnOrigins {
		if err := validateAuthHTTPURL("AUTH_ALLOWED_RETURN_ORIGINS", origin, production, true); err != nil {
			return err
		}
	}
	if c.Adapter == "dev" {
		if production {
			return fmt.Errorf("AUTH_LOGIN_ADAPTER=dev is forbidden in production")
		}
		return nil
	}
	if c.Adapter != "oidc" {
		return fmt.Errorf("AUTH_LOGIN_ADAPTER must be oidc or dev")
	}
	if c.AuthorizationURL == "" || c.TokenURL == "" || c.ClientID == "" || c.RedirectURL == "" {
		return fmt.Errorf("Configurator OIDC login configuration is incomplete")
	}
	for key, value := range map[string]string{
		"AUTH_AUTHORIZATION_URL": c.AuthorizationURL,
		"AUTH_TOKEN_URL":         c.TokenURL,
		"AUTH_REDIRECT_URL":      c.RedirectURL,
	} {
		if err := validateAuthHTTPURL(key, value, production, false); err != nil {
			return err
		}
	}
	if c.SessionTTL <= 0 || c.TransactionTTL <= 0 || c.SessionCleanupInterval <= 0 {
		return fmt.Errorf("auth session durations must be positive")
	}
	if production && !c.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in production")
	}
	return nil
}

func validateAuthHTTPURL(key, value string, production, originOnly bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%s must contain absolute HTTP URLs without user info", key)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", key)
	}
	if originOnly && ((parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "") {
		return fmt.Errorf("%s entries must contain only scheme, host and optional port", key)
	}
	if production && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", key)
	}
	return nil
}

func parseEncryptionKeys(value string) ([]EncryptionKeyConfig, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	result := make([]EncryptionKeyConfig, 0)
	for _, raw := range strings.Split(value, ",") {
		parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("ENCRYPTION_PREVIOUS_KEYS must use key-id:base64-key entries")
		}
		result = append(result, EncryptionKeyConfig{ID: strings.TrimSpace(parts[0]), Key: strings.TrimSpace(parts[1])})
	}
	return result, nil
}

func (c EncryptionConfig) Validate() error {
	decoded, err := base64.StdEncoding.DecodeString(c.Key)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	if !validEncryptionKeyID(c.KeyID) {
		return fmt.Errorf("ENCRYPTION_KEY_ID must contain only letters, digits, dot, underscore or hyphen")
	}
	seen := map[string]struct{}{c.KeyID: {}}
	for _, previous := range c.PreviousKeys {
		if !validEncryptionKeyID(previous.ID) {
			return fmt.Errorf("previous encryption key id %q is invalid", previous.ID)
		}
		if _, exists := seen[previous.ID]; exists {
			return fmt.Errorf("encryption key id %q is duplicated", previous.ID)
		}
		seen[previous.ID] = struct{}{}
		decoded, decodeErr := base64.StdEncoding.DecodeString(previous.Key)
		if decodeErr != nil || len(decoded) != 32 {
			return fmt.Errorf("previous encryption key %q must be a base64-encoded 32-byte key", previous.ID)
		}
	}
	return nil
}

func validEncryptionKeyID(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func legacyJWKSURL(value kitconfig.ServiceAuthConfig) string {
	if strings.TrimSpace(value.ServiceURL) == "" {
		return ""
	}
	return value.JWKSURL()
}

func (c IdentityConfig) Validate(production bool) error {
	switch c.Mode {
	case "oidc":
		if c.ProviderID == "" || c.Issuer == "" || c.JWKSURL == "" || len(c.AllowedAudiences) == 0 || len(c.AllowedAlgorithms) == 0 {
			return fmt.Errorf("OIDC auth configuration is incomplete")
		}
		for _, algorithm := range c.AllowedAlgorithms {
			upper := strings.ToUpper(strings.TrimSpace(algorithm))
			if upper == "NONE" || strings.HasPrefix(upper, "HS") {
				return fmt.Errorf("AUTH_ALLOWED_ALGORITHMS contains forbidden algorithm %q", algorithm)
			}
		}
	case "dev":
		if production {
			return fmt.Errorf("AUTH_MODE=dev is forbidden in production")
		}
		if c.DevSubject == "" {
			return fmt.Errorf("AUTH_DEV_SUBJECT is required in dev mode")
		}
	default:
		return fmt.Errorf("AUTH_MODE must be oidc or dev")
	}
	return nil
}

func modeDefault(production bool) string {
	if production {
		return "oidc"
	}
	return "dev"
}

func loginAdapterDefault(identityMode string) string {
	if identityMode == "dev" {
		return "dev"
	}
	return "oidc"
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func csv(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func envBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func envIntAllowZero(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
