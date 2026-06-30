package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/platform"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newPostgres(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	return platform.NewPostgresPool(lc, cfg, logger)
}
