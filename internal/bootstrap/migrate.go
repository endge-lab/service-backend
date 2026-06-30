package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/platform"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Migrate(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) {
	if err := platform.RunMigrations(lc, cfg, logger); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
}
