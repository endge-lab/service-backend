package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/endge-lab/service-backend/internal/usecase/actions"
	"github.com/endge-lab/service-backend/internal/usecase/ai_assistant"
	"github.com/endge-lab/service-backend/internal/usecase/ai_catalog"
	"github.com/endge-lab/service-backend/internal/usecase/auth_profiles"
	"github.com/endge-lab/service-backend/internal/usecase/backend_connections"
	"github.com/endge-lab/service-backend/internal/usecase/backups"
	"github.com/endge-lab/service-backend/internal/usecase/commits"
	"github.com/endge-lab/service-backend/internal/usecase/components"
	"github.com/endge-lab/service-backend/internal/usecase/compositions"
	"github.com/endge-lab/service-backend/internal/usecase/computations"
	"github.com/endge-lab/service-backend/internal/usecase/configurations"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/environments"
	"github.com/endge-lab/service-backend/internal/usecase/filters"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/i18n_bundles"
	"github.com/endge-lab/service-backend/internal/usecase/integrations"
	"github.com/endge-lab/service-backend/internal/usecase/mocks"
	"github.com/endge-lab/service-backend/internal/usecase/navigations"
	"github.com/endge-lab/service-backend/internal/usecase/portable"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
	"github.com/endge-lab/service-backend/internal/usecase/release_artifacts"
	"github.com/endge-lab/service-backend/internal/usecase/releases"
	"github.com/endge-lab/service-backend/internal/usecase/revisions"
	"github.com/endge-lab/service-backend/internal/usecase/service_info"
	"github.com/endge-lab/service-backend/internal/usecase/session"
	"github.com/endge-lab/service-backend/internal/usecase/stores"
	"github.com/endge-lab/service-backend/internal/usecase/streams"
	"github.com/endge-lab/service-backend/internal/usecase/styles"
	"github.com/endge-lab/service-backend/internal/usecase/tenants"
	"github.com/endge-lab/service-backend/internal/usecase/types"
	"github.com/endge-lab/service-backend/internal/usecase/updates"
	"github.com/endge-lab/service-backend/internal/usecase/vocabs"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
	"go.uber.org/fx"
)

func UseCaseModules() fx.Option {
	return fx.Options(fx.Provide(
		releaseArtifactCacheConfig,
		newWorkspaceStateCoordinator,
		history.NewRecorder,
		documents.NewLifecycle,
		access_control.NewUseCase,
		ai_catalog.NewUseCase,
		ai_assistant.NewUseCase,
		service_info.NewUseCase,
		workspaces.NewUseCase,
		backend_connections.NewUseCase,
		integrations.NewUseCase,
		session.NewUseCase,
		projects.NewUseCase,
		tenants.NewUseCase,
		environments.NewUseCase,
		folders.NewUseCase,
		types.NewUseCase,
		queries.NewUseCase,
		data_views.NewUseCase,
		compositions.NewUseCase,
		stores.NewUseCase,
		streams.NewUseCase,
		updates.NewUseCase,
		mocks.NewUseCase,
		components.NewUseCase,
		actions.NewUseCase,
		filters.NewUseCase,
		converters.NewUseCase,
		computations.NewUseCase,
		vocabs.NewUseCase,
		i18n_bundles.NewUseCase,
		auth_profiles.NewUseCase,
		navigations.NewUseCase,
		styles.NewUseCase,
		configurations.NewUseCase,
		revisions.NewUseCase,
		commits.NewUseCase,
		portable.NewUseCase,
		fx.Annotate(release_artifacts.NewReader, fx.As(new(ports.ReleaseArtifactReader))),
		releases.NewUseCase,
		backups.NewUseCase,
	))
}

func releaseArtifactCacheConfig(cfg *config.Config) config.ReleaseArtifactCacheConfig {
	return cfg.ReleaseArtifactCache
}

func newWorkspaceStateCoordinator(repository workspace_state.Repository, tx ports.TxManager, artifacts ports.ReleaseArtifactReader, cfg *config.Config) *workspace_state.Coordinator {
	return workspace_state.NewCoordinator(repository, tx, artifacts, cfg.WorkspaceSchemaVersion)
}
