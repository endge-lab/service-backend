package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

type endgeRepositoryPorts struct {
	fx.Out

	WorkspaceState     workspace_state.Repository
	Workspaces         ports.WorkspaceRepository
	Integrations       ports.IntegrationRepository
	BackendConnections ports.BackendConnectionRepository
	AccessControl      ports.AccessControlRepository
	Documents          ports.DocumentRepository
	Revisions          ports.RevisionRepository
	Commits            ports.CommitRepository
	Releases           ports.ReleaseRepository
	ReleaseArtifacts   ports.ReleaseArtifactRepository
	Portable           ports.PortableRepository
	Snapshots          ports.SnapshotRepository
}

func exposeEndgeRepository(store *postgres.EndgeRepository) endgeRepositoryPorts {
	return endgeRepositoryPorts{
		WorkspaceState:     store,
		Workspaces:         store,
		Integrations:       store,
		BackendConnections: store,
		AccessControl:      store,
		Documents:          store,
		Revisions:          store,
		Commits:            store,
		Releases:           store,
		ReleaseArtifacts:   store,
		Portable:           store,
		Snapshots:          store,
	}
}

func RepositoryModules() fx.Option {
	return fx.Options(fx.Provide(
		postgres.NewRepositoryMetrics,
		func(db *pgxpool.Pool) *sqlc.Queries { return sqlc.New(db) },
		fx.Annotate(postgres.NewUserRepository, fx.As(new(ports.UserRepository))),
		postgres.NewEndgeRepository,
		exposeEndgeRepository,
		postgres.NewProjectRepository,
		postgres.NewTenantRepository,
		postgres.NewEnvironmentRepository,
		postgres.NewFolderRepository,
		postgres.NewTypeRepository,
		postgres.NewQueryRepository,
		postgres.NewDataViewRepository,
		postgres.NewCompositionRepository,
		postgres.NewStoreRepository,
		postgres.NewStreamRepository,
		postgres.NewUpdateRepository,
		postgres.NewMockRepository,
		postgres.NewComponentRepository,
		postgres.NewActionRepository,
		postgres.NewFilterRepository,
		postgres.NewConverterRepository,
		postgres.NewComputationRepository,
		postgres.NewVocabRepository,
		postgres.NewI18nBundleRepository,
		postgres.NewAuthProfileRepository,
		postgres.NewNavigationRepository,
		postgres.NewStyleRepository,
		postgres.NewConfigurationRepository,
		fx.Annotate(postgres.NewTxManager, fx.As(new(ports.TxManager))),
	))
}
