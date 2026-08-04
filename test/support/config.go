//go:build integration || e2e

package support

import (
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	kitconfig "github.com/endge-lab/service-kit-go/config"
)

// DevConfig создаёт полностью явную конфигурацию без обращения к process env.
func DevConfig() *config.Config {
	base := &kitconfig.ServiceConfig{
		App:       kitconfig.ServiceAppConfig{Env: "test", Name: "service-backend-test", Version: "test", PublicURL: "http://backend.test"},
		HTTP:      kitconfig.ServiceHTTPConfig{Port: "0", CORSAllowedOrigins: "http://configurator.test"},
		Logger:    kitconfig.ServiceLoggerConfig{Level: "error"},
		Metrics:   kitconfig.ServiceMetricsConfig{Enabled: false},
		Telemetry: kitconfig.ServiceTelemetryConfig{Enabled: false},
		Redpanda:  kitconfig.ServiceRedpandaConfig{Enabled: false},
		Postgres:  kitconfig.ServicePostgresConfig{Schema: "public", SSLMode: "disable", MigrationsEnabled: false},
	}
	return &config.Config{
		ServiceConfig:    base,
		Identity:         config.IdentityConfig{Mode: "dev", DevSubject: "e2e-user", DevUsername: "e2e", DevDisplayName: "E2E User", DevPlatformAdmin: true},
		ConfiguratorAuth: config.ConfiguratorAuthConfig{Adapter: "dev", ReturnURL: "http://configurator.test", SessionCookieName: "endge_test_session", SessionTTL: time.Hour, TransactionTTL: time.Minute},
		Snapshots:        config.SnapshotConfig{ImportBackupRetentionDays: 7},
	}
}
