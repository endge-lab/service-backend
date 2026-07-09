package usecase

import (
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
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
