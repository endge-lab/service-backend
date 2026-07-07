package repo

import (
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

type repositoryFactory struct {
	deps Params
}

func newRepositoryFactory(deps Params) *repositoryFactory {
	return &repositoryFactory{deps: deps}
}

func (f *repositoryFactory) BuildUserRepository() ports.UserRepository {
	return postgres.NewUserRepository(f.deps.DB, sqlc.New(f.deps.DB), f.deps.Tracer, f.deps.Logger)
}

func (f *repositoryFactory) BuildTodoRepository() ports.TodoRepository {
	return postgres.NewTodoRepository(f.deps.DB, sqlc.New(f.deps.DB), f.deps.Tracer, f.deps.Logger)
}

func (f *repositoryFactory) BuildTxManager() ports.TxManager {
	return postgres.NewTxManager(f.deps.DB, f.deps.Tracer, f.deps.Logger)
}

func (f *repositoryFactory) BuildProjectsRepository() ports.ProjectsRepository {
	return postgres.NewProjectsRepository(
		f.deps.DB, sqlc.New(f.deps.DB), f.deps.Logger, f.deps.Tracer)
}
