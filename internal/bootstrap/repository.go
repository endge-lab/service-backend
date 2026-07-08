package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func RepositoryModules() fx.Option {
	return fx.Options(
		fx.Provide(
			func(db *pgxpool.Pool) *sqlc.Queries {
				return sqlc.New(db)
			},
			fx.Annotate(
				postgres.NewUserRepository,
				fx.As(new(ports.UserRepository)),
			),
			fx.Annotate(
				postgres.NewProjectsRepository,
				fx.As(new(ports.ProjectsRepository)),
			),
			fx.Annotate(
				postgres.NewTxManager,
				fx.As(new(ports.TxManager)),
			),
		),
	)
}
