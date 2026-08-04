package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	Adapter              string
	AuthorizationURL     string
	TokenURL             string
	LogoutURL            string
	ClientID             string
	ClientSecret         string
	Scopes               []string
	RedirectURL          string
	ReturnURL            string
	SessionCookieName    string
	SessionEncryptionKey string
	SessionTTL           time.Duration
	TransactionTTL       time.Duration
	CookieSecure         bool
	CookieDomain         string
}

type Config struct {
	*kitconfig.ServiceConfig
	Identity         IdentityConfig
	ConfiguratorAuth ConfiguratorAuthConfig
	Snapshots        SnapshotConfig
}

// SnapshotConfig задаёт срок хранения временных страховочных копий импорта.
type SnapshotConfig struct {
	ImportBackupRetentionDays int
}

func Load() (*Config, error) {
	base, err := kitconfig.LoadServiceConfig()
	if err != nil {
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
	configuratorAuth := ConfiguratorAuthConfig{
		Adapter:              env("AUTH_LOGIN_ADAPTER", loginAdapterDefault(identity.Mode)),
		AuthorizationURL:     env("AUTH_AUTHORIZATION_URL", ""),
		TokenURL:             env("AUTH_TOKEN_URL", ""),
		LogoutURL:            env("AUTH_LOGOUT_URL", ""),
		ClientID:             env("AUTH_CLIENT_ID", ""),
		ClientSecret:         strings.TrimSpace(os.Getenv("AUTH_CLIENT_SECRET")),
		Scopes:               csv(env("AUTH_SCOPES", "openid,profile,email")),
		RedirectURL:          env("AUTH_REDIRECT_URL", strings.TrimRight(base.App.PublicURL, "/")+"/auth/callback"),
		ReturnURL:            env("AUTH_RETURN_URL", strings.TrimRight(base.App.PublicURL, "/")),
		SessionCookieName:    env("AUTH_SESSION_COOKIE_NAME", "endge_configurator_session"),
		SessionEncryptionKey: strings.TrimSpace(os.Getenv("AUTH_SESSION_ENCRYPTION_KEY")),
		SessionTTL:           envDuration("AUTH_SESSION_TTL", 8*time.Hour),
		TransactionTTL:       envDuration("AUTH_TRANSACTION_TTL", 10*time.Minute),
		CookieSecure:         envBool("AUTH_COOKIE_SECURE", base.App.IsProduction()),
		CookieDomain:         strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
	}
	if err := configuratorAuth.Validate(base.App.IsProduction()); err != nil {
		return nil, err
	}
	return &Config{
		ServiceConfig:    base,
		Identity:         identity,
		ConfiguratorAuth: configuratorAuth,
		Snapshots: SnapshotConfig{
			ImportBackupRetentionDays: envInt("IMPORT_BACKUP_RETENTION_DAYS", 7),
		},
	}, nil
}

func (c ConfiguratorAuthConfig) Validate(production bool) error {
	if c.Adapter == "dev" {
		if production {
			return fmt.Errorf("AUTH_LOGIN_ADAPTER=dev is forbidden in production")
		}
		return nil
	}
	if c.Adapter != "oidc" {
		return fmt.Errorf("AUTH_LOGIN_ADAPTER must be oidc or dev")
	}
	if c.AuthorizationURL == "" || c.TokenURL == "" || c.ClientID == "" || c.RedirectURL == "" || c.ReturnURL == "" {
		return fmt.Errorf("Configurator OIDC login configuration is incomplete")
	}
	for key, value := range map[string]string{
		"AUTH_AUTHORIZATION_URL": c.AuthorizationURL,
		"AUTH_TOKEN_URL":         c.TokenURL,
		"AUTH_REDIRECT_URL":      c.RedirectURL,
		"AUTH_RETURN_URL":        c.ReturnURL,
	} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("%s must be an absolute URL", key)
		}
		if production && parsed.Scheme != "https" {
			return fmt.Errorf("%s must use https in production", key)
		}
	}
	key, err := base64.StdEncoding.DecodeString(c.SessionEncryptionKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("AUTH_SESSION_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	if c.SessionTTL <= 0 || c.TransactionTTL <= 0 {
		return fmt.Errorf("auth session durations must be positive")
	}
	if production && !c.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in production")
	}
	return nil
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

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
