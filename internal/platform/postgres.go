package platform

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/endge-lab/service-template-go/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewPostgresPool(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *zap.Logger,
) (*pgxpool.Pool, error) {
	parsedConfig, err := pgxpool.ParseConfig(cfg.Postgres.URI)
	if err != nil {
		logger.Error("pgx parse config failed", zap.Error(err))
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), parsedConfig)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL", zap.Error(err))
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing postgres pool")
			pool.Close()
			return nil
		},
	})

	logger.Info("connected to PostgreSQL")
	return pool, nil
}

func RunMigrations(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) error {
	db, err := sql.Open("postgres", cfg.Postgres.URI)
	if err != nil {
		return fmt.Errorf("failed to open DB for migration: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("migrations applied")

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})

	return nil
}
