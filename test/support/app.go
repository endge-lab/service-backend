//go:build e2e

package support

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/bootstrap"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// NewTestApp собирает настоящий HTTP pipeline, но подменяет только config и pool.
// Environment-файлы при этом не читаются, TCP listener не запускается.
func NewTestApp(t testing.TB, database *TestDatabase, cfg *config.Config) *fiber.App {
	t.Helper()
	var fiberApp *fiber.App
	application := bootstrap.NewApp(
		fx.Replace(cfg, database.Pool),
		fx.Populate(&fiberApp),
		fx.NopLogger,
	)
	if err := application.Err(); err != nil {
		t.Fatalf("собрать тестовое приложение: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := application.Stop(ctx); err != nil {
			t.Errorf("остановить тестовое приложение: %v", err)
		}
	})
	return fiberApp
}

// OIDCConfig включает реальную bearer-проверку через локальный JWKS provider.
func OIDCConfig(provider *IdentityProvider) *config.Config {
	value := DevConfig()
	value.Identity = config.IdentityConfig{
		Mode: "oidc", ProviderID: "test-oidc", Issuer: provider.URL(), JWKSURL: provider.URL() + "/jwks",
		AllowedAudiences: []string{"endge-configurator"}, AllowedAlgorithms: []string{"RS256"},
		UsernameClaim: "preferred_username", DisplayNameClaim: "name", GroupsClaim: "groups",
		PlatformAdminGroups: []string{"endge-platform-admins"},
	}
	value.ConfiguratorAuth = config.ConfiguratorAuthConfig{
		Adapter: "oidc", AuthorizationURL: provider.URL() + "/authorize", TokenURL: provider.URL() + "/token",
		LogoutURL: provider.URL() + "/logout", ClientID: "endge-configurator", RedirectURL: "http://backend.test/auth/callback",
		ReturnURL: "http://configurator.test", SessionCookieName: "endge_test_session",
		SessionEncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), SessionTTL: time.Hour,
		SessionEncryptionKeyID: "test-v1", TransactionTTL: time.Minute, SessionCleanupInterval: time.Minute,
		Scopes: []string{"openid", "profile"},
	}
	return value
}
