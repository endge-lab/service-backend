package bootstrap

import (
	"context"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/platform"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func InitLogger(lc fx.Lifecycle, cfg *config.Config) *zap.Logger {
	logger := platform.NewLogger(cfg.LoggerLevel, cfg.AppName, cfg.AppEnv, cfg.AppVersion)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			_ = logger.Sync()
			return nil
		},
	})

	return logger
}
