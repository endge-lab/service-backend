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
			postgres.NewRepositoryMetrics,
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
				postgres.NewWorkspacesRepository,
				fx.As(new(ports.WorkspacesRepository)),
			),
			fx.Annotate(
				postgres.NewTenantsRepository,
				fx.As(new(ports.TenantsRepository)),
			),
			fx.Annotate(
				postgres.NewFoldersRepository,
				fx.As(new(ports.FoldersRepository)),
			),
			fx.Annotate(
				postgres.NewComponentsLegacyRepository,
				fx.As(new(ports.ComponentsLegacyRepository)),
			),
			fx.Annotate(
				postgres.NewConvertersRepository,
				fx.As(new(ports.ConvertersRepository)),
			),
			fx.Annotate(
				postgres.NewQueriesRepository,
				fx.As(new(ports.QueriesRepository)),
			),
			fx.Annotate(
				postgres.NewDataViewsRepository,
				fx.As(new(ports.DataViewsRepository)),
			),
			fx.Annotate(
				postgres.NewDomainDependenciesRepository,
				fx.As(new(ports.DomainDependenciesRepository)),
			),
			fx.Annotate(
				postgres.NewTxManager,
				fx.As(new(ports.TxManager)),
			),
		),
	)
}
