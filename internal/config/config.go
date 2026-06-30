package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type TelemetryConfig struct {
	Enabled      bool   `envconfig:"TELEMETRY_ENABLED" default:"false"`
	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
	OTLPInsecure bool   `envconfig:"OTEL_EXPORTER_OTLP_INSECURE" default:"false"`
}

type RedpandaConfig struct {
	Enabled          bool          `envconfig:"REDPANDA_ENABLED" default:"false"`
	Brokers          string        `envconfig:"REDPANDA_BROKERS" default:""`
	ClientID         string        `envconfig:"REDPANDA_CLIENT_ID" default:""`
	DialTimeout      time.Duration `envconfig:"REDPANDA_DIAL_TIMEOUT" default:"5s"`
	ReadBatchTimeout time.Duration `envconfig:"REDPANDA_READ_BATCH_TIMEOUT" default:"2s"`
	WriteTimeout     time.Duration `envconfig:"REDPANDA_WRITE_TIMEOUT" default:"10s"`
}

type AuthConfig struct {
	Enabled          bool          `envconfig:"AUTH_ENABLED" default:"false"`
	ServiceURL       string        `envconfig:"AUTH_SERVICE_URL" default:""`
	Issuer           string        `envconfig:"AUTH_ISSUER" default:""`
	AllowedAudiences string        `envconfig:"AUTH_ALLOWED_AUDIENCES" default:""`
	JWKSPath         string        `envconfig:"AUTH_JWKS_PATH" default:"/.well-known/jwks.json"`
	JWKSCacheTTL     time.Duration `envconfig:"AUTH_JWKS_CACHE_TTL" default:"5m"`
	Timeout          time.Duration `envconfig:"AUTH_SERVICE_TIMEOUT" default:"5s"`
}

type PostgresConfig struct {
	URI string `envconfig:"DATABASE_URI" default:""`
}

type Config struct {
	AppEnv             string `envconfig:"APP_ENV" default:"development"`
	AppName            string `envconfig:"APP_NAME" default:""`
	AppVersion         string `envconfig:"APP_VERSION" default:"dev"`
	RestPort           string `envconfig:"REST_PORT" default:"8080"`
	LoggerLevel        string `envconfig:"LOGGER_LEVEL" default:"debug"`
	PublicURL          string `envconfig:"PUBLIC_URL" default:""`
	CORSAllowedOrigins string `envconfig:"CORS_ALLOWED_ORIGINS" default:""`
	Auth               AuthConfig
	Postgres           PostgresConfig
	Telemetry          TelemetryConfig
	Redpanda           RedpandaConfig
}

func detectAppEnv() string {
	for _, key := range []string{"APP_ENV", "GO_ENV", "ENVIRONMENT", "NODE_ENV"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return "development"
}

func normalizeDeploymentEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "dev", "development":
		return "development"
	case "local", "dev.local", "development.local", "local.development":
		return "development.local"
	case "prod", "production":
		return "production"
	default:
		return strings.TrimSpace(value)
	}
}

func preloadEnvFiles() {
	appEnv := detectAppEnv()
	searchDirs := []string{"."}
	seen := map[string]struct{}{}

	if executable, err := os.Executable(); err == nil {
		searchDirs = append(searchDirs, filepath.Dir(executable))
	}

	for _, dir := range searchDirs {
		for _, fileName := range envFileCandidates(appEnv) {
			filePath := filepath.Join(dir, fileName)
			if _, exists := seen[filePath]; exists {
				continue
			}
			seen[filePath] = struct{}{}

			if _, err := os.Stat(filePath); err == nil {
				_ = godotenv.Load(filePath)
			}
		}
	}
}

func envFileCandidates(appEnv string) []string {
	trimmedAppEnv := strings.TrimSpace(appEnv)
	normalizedAppEnv := normalizeDeploymentEnvironment(appEnv)

	switch normalizedAppEnv {
	case "development", "development.local":
		return []string{".env.development.local", ".env.local", ".env.development", ".env"}
	case "production":
		return []string{".env.production", ".env.local", ".env"}
	default:
		if trimmedAppEnv == "" {
			return []string{".env.development.local", ".env.local", ".env.development", ".env"}
		}
		return []string{".env." + trimmedAppEnv + ".local", ".env.local", ".env." + trimmedAppEnv, ".env"}
	}
}

func Load() *Config {
	var cfg Config

	preloadEnvFiles()

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("не удалось загрузить конфигурацию: %v", err)
	}

	cfg.AppEnv = normalizeDeploymentEnvironment(cfg.AppEnv)

	if cfg.Auth.Timeout <= 0 {
		cfg.Auth.Timeout = 5 * time.Second
	}
	if cfg.Auth.JWKSCacheTTL <= 0 {
		cfg.Auth.JWKSCacheTTL = 5 * time.Minute
	}

	cfg.Auth.ServiceURL = strings.TrimRight(strings.TrimSpace(cfg.Auth.ServiceURL), "/")
	cfg.Auth.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Auth.Issuer), "/")
	cfg.Auth.JWKSPath = ensureLeadingSlash(strings.TrimSpace(cfg.Auth.JWKSPath))
	cfg.Auth.AllowedAudiences = strings.TrimSpace(cfg.Auth.AllowedAudiences)
	cfg.Postgres.URI = strings.TrimSpace(cfg.Postgres.URI)
	cfg.PublicURL = strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/")
	cfg.CORSAllowedOrigins = strings.TrimSpace(cfg.CORSAllowedOrigins)
	cfg.Telemetry.OTLPEndpoint = strings.TrimSpace(cfg.Telemetry.OTLPEndpoint)
	cfg.Redpanda.Brokers = strings.TrimSpace(cfg.Redpanda.Brokers)

	if cfg.Redpanda.DialTimeout <= 0 {
		cfg.Redpanda.DialTimeout = 5 * time.Second
	}
	if cfg.Redpanda.ReadBatchTimeout <= 0 {
		cfg.Redpanda.ReadBatchTimeout = 2 * time.Second
	}
	if cfg.Redpanda.WriteTimeout <= 0 {
		cfg.Redpanda.WriteTimeout = 10 * time.Second
	}
	if strings.TrimSpace(cfg.Redpanda.ClientID) == "" {
		cfg.Redpanda.ClientID = cfg.AppName
	}

	switch {
	case cfg.AppName == "":
		log.Fatal("не удалось загрузить конфигурацию: APP_NAME is required")
	case cfg.PublicURL == "":
		log.Fatal("не удалось загрузить конфигурацию: PUBLIC_URL is required")
	case cfg.CORSAllowedOrigins == "":
		log.Fatal("не удалось загрузить конфигурацию: CORS_ALLOWED_ORIGINS is required")
	case cfg.Auth.Enabled && cfg.Auth.ServiceURL == "":
		log.Fatal("не удалось загрузить конфигурацию: AUTH_SERVICE_URL is required")
	case cfg.Auth.Enabled && cfg.Auth.Issuer == "":
		log.Fatal("не удалось загрузить конфигурацию: AUTH_ISSUER is required")
	case cfg.Postgres.URI == "":
		log.Fatal("не удалось загрузить конфигурацию: DATABASE_URI is required")
	case cfg.Telemetry.Enabled && cfg.Telemetry.OTLPEndpoint == "":
		log.Fatal("не удалось загрузить конфигурацию: OTEL_EXPORTER_OTLP_ENDPOINT is required")
	case cfg.Redpanda.Enabled && len(cfg.Redpanda.BrokerList()) == 0:
		log.Fatal("не удалось загрузить конфигурацию: REDPANDA_BROKERS is required when REDPANDA_ENABLED=true")
	}

	return &cfg
}

func ensureLeadingSlash(value string) string {
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func (c AuthConfig) JWKSURL() string {
	return strings.TrimRight(c.ServiceURL, "/") + ensureLeadingSlash(c.JWKSPath)
}

func (c RedpandaConfig) BrokerList() []string {
	if strings.TrimSpace(c.Brokers) == "" {
		return nil
	}

	parts := strings.Split(c.Brokers, ",")
	brokers := make([]string, 0, len(parts))

	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker == "" {
			continue
		}
		brokers = append(brokers, broker)
	}

	return brokers
}
