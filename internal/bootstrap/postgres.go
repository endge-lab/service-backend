package bootstrap

import (
	"context"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/migrations"
	medb "github.com/endge-lab/service-kit-go/pkg/db/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newPostgres(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	return medb.NewPostgresClient(
		context.Background(),
		lc,
		cfg.Postgres,
		logger,
		medb.WithEmbedFS(migrations.FS),
	)
}
