package bootstrap

import (
	httpsession "github.com/endge-lab/service-backend/internal/api/http/session"
	httpcomponentlegacy "github.com/endge-lab/service-backend/internal/api/http/v1/component_legacy"
	httpconverter "github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	httpdataview "github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	httpfolder "github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	httpproject "github.com/endge-lab/service-backend/internal/api/http/v1/project"
	httpquery "github.com/endge-lab/service-backend/internal/api/http/v1/query"
	httpworkspace "github.com/endge-lab/service-backend/internal/api/http/v1/workspace"
	"github.com/endge-lab/service-backend/internal/usecase/components_legacy"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
	usecasesession "github.com/endge-lab/service-backend/internal/usecase/session"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func UseCaseModules() fx.Option {
	return fx.Options(
		fx.Provide(
			shared.NewUseCaseMetrics,
			fx.Annotate(usecasesession.NewLoadSessionUseCase, fx.As(new(httpsession.UseCase))),
			fx.Annotate(newProjectUseCase, fx.As(new(httpproject.UseCase))),
			fx.Annotate(newFolderUseCase, fx.As(new(httpfolder.UseCase))),
			fx.Annotate(newComponentLegacyUseCase, fx.As(new(httpcomponentlegacy.UseCase))),
			fx.Annotate(newConverterUseCase, fx.As(new(httpconverter.UseCase))),
			fx.Annotate(newQueryUseCase, fx.As(new(httpquery.UseCase))),
			fx.Annotate(newDataViewUseCase, fx.As(new(httpdataview.UseCase))),
			fx.Annotate(newWorkspaceUseCase, fx.As(new(httpworkspace.UseCase))),
		),
	)
}

func newWorkspaceUseCase(repository ports.WorkspacesRepository, tracer trace.Tracer, logger *zap.Logger, metrics *shared.UseCaseMetrics) *workspaces.Workspace {
	return workspaces.NewWorkspaceService(workspaces.WorkspaceParams{Repository: repository, Tracer: tracer, Logger: logger, Metrics: metrics})
}

func newProjectUseCase(
	projectRepository ports.ProjectsRepository,
	folderRepository ports.FoldersRepository,
	txManager ports.TxManager,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *projects.Project {
	return projects.NewProjectService(projects.ProjectParams{
		ProjectRepository: projectRepository,
		FolderRepository:  folderRepository,
		TxManager:         txManager,
		Tracer:            tracer,
		Logger:            logger,
		Metrics:           metrics,
	})
}

func newFolderUseCase(
	folderRepository ports.FoldersRepository,
	projectRepository ports.ProjectsRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *folders.Folder {
	return folders.NewFolderService(folders.FolderParams{
		FolderRepository:  folderRepository,
		ProjectRepository: projectRepository,
		Tracer:            tracer,
		Logger:            logger,
		Metrics:           metrics,
	})
}

func newComponentLegacyUseCase(
	componentRepository ports.ComponentsLegacyRepository,
	folderRepository ports.FoldersRepository,
	projectRepository ports.ProjectsRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *components_legacy.ComponentLegacy {
	return components_legacy.NewComponentLegacyService(components_legacy.ComponentLegacyParams{
		ComponentLegacyRepository: componentRepository,
		FolderRepository:          folderRepository,
		ProjectRepository:         projectRepository,
		Tracer:                    tracer,
		Logger:                    logger,
		Metrics:                   metrics,
	})
}

func newConverterUseCase(
	converterRepository ports.ConvertersRepository,
	folderRepository ports.FoldersRepository,
	projectRepository ports.ProjectsRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *converters.Converter {
	return converters.NewConverterService(converters.ConverterParams{
		ConverterRepository: converterRepository,
		FolderRepository:    folderRepository,
		ProjectRepository:   projectRepository,
		Tracer:              tracer,
		Logger:              logger,
		Metrics:             metrics,
	})
}

func newQueryUseCase(
	queryRepository ports.QueriesRepository,
	folderRepository ports.FoldersRepository,
	projectRepository ports.ProjectsRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *queries.Query {
	return queries.NewQueryService(queries.QueryParams{
		QueryRepository:   queryRepository,
		FolderRepository:  folderRepository,
		ProjectRepository: projectRepository,
		Tracer:            tracer,
		Logger:            logger,
		Metrics:           metrics,
	})
}

func newDataViewUseCase(
	dataViewRepository ports.DataViewsRepository,
	queryRepository ports.QueriesRepository,
	folderRepository ports.FoldersRepository,
	projectRepository ports.ProjectsRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) *data_views.DataView {
	return data_views.NewDataViewService(data_views.DataViewParams{
		DataViewRepository: dataViewRepository,
		QueryRepository:    queryRepository,
		FolderRepository:   folderRepository,
		ProjectRepository:  projectRepository,
		Tracer:             tracer,
		Logger:             logger,
		Metrics:            metrics,
	})
}
