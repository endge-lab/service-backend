package usecase

import (
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/components"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
)

type serviceFactory struct {
	deps Params
}

func newServiceFactory(deps Params) *serviceFactory {
	return &serviceFactory{deps: deps}
}

func (f *serviceFactory) CreateLoadSessionUseCase() adapters.LoadSessionService {
	return newLoadSessionUseCase(LoadSessionParams{
		UserRepository: f.deps.UserRepository,
		Tracer:         f.deps.Tracer,
		Logger:         f.deps.Logger,
		Metrics:        f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateProjectsUseCase() adapters.ProjectService {
	return projects.NewProjectService(projects.ProjectParams{
		ProjectRepository: f.deps.ProjectRepository,
		FolderRepository:  f.deps.FolderRepository,
		TxManager:         f.deps.TxManager,
		Tracer:            f.deps.Tracer,
		Logger:            f.deps.Logger,
		Metrics:           f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateFoldersUseCase() adapters.FolderService {
	return folders.NewFolderService(folders.FolderParams{
		FolderRepository:  f.deps.FolderRepository,
		ProjectRepository: f.deps.ProjectRepository,
		Tracer:            f.deps.Tracer,
		Logger:            f.deps.Logger,
		Metrics:           f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateComponentsUseCase() adapters.ComponentService {
	return components.NewComponentService(components.ComponentParams{
		ComponentRepository: f.deps.ComponentRepository,
		FolderRepository:    f.deps.FolderRepository,
		ProjectRepository:   f.deps.ProjectRepository,
		Tracer:              f.deps.Tracer,
		Logger:              f.deps.Logger,
		Metrics:             f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateConvertersUseCase() adapters.ConverterService {
	return converters.NewConverterService(converters.ConverterParams{
		ConverterRepository: f.deps.ConverterRepository,
		FolderRepository:    f.deps.FolderRepository,
		ProjectRepository:   f.deps.ProjectRepository,
		Tracer:              f.deps.Tracer,
		Logger:              f.deps.Logger,
		Metrics:             f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateQueriesUseCase() adapters.QueryService {
	return queries.NewQueryService(queries.QueryParams{
		QueryRepository:   f.deps.QueryRepository,
		FolderRepository:  f.deps.FolderRepository,
		ProjectRepository: f.deps.ProjectRepository,
		Tracer:            f.deps.Tracer,
		Logger:            f.deps.Logger,
		Metrics:           f.deps.Metrics,
	})
}

func (f *serviceFactory) CreateDataViewsUseCase() adapters.DataViewService {
	return data_views.NewDataViewService(data_views.DataViewParams{
		DataViewRepository: f.deps.DataViewRepository,
		QueryRepository:    f.deps.QueryRepository,
		FolderRepository:   f.deps.FolderRepository,
		ProjectRepository:  f.deps.ProjectRepository,
		Tracer:             f.deps.Tracer,
		Logger:             f.deps.Logger,
		Metrics:            f.deps.Metrics,
	})
}
