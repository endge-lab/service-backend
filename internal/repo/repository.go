package repo

import (
	"github.com/endge-lab/service-backend/internal/repo/ports"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Repository struct {
	user      ports.UserRepository
	todo      ports.TodoRepository
	txManager ports.TxManager
	project   ports.ProjectsRepository
}

type Params struct {
	fx.In

	Logger *zap.Logger
	DB     *pgxpool.Pool
	Tracer trace.Tracer
}

func NewRepository(params Params) *Repository {
	factory := newRepositoryFactory(params)

	return &Repository{
		user:      factory.BuildUserRepository(),
		todo:      factory.BuildTodoRepository(),
		txManager: factory.BuildTxManager(),
		project:   factory.BuildProjectsRepository(),
	}
}

func (r *Repository) User() ports.UserRepository {
	return r.user
}

func (r *Repository) Todo() ports.TodoRepository {
	return r.todo
}

func (r *Repository) TxManager() ports.TxManager {
	return r.txManager
}
func (r *Repository) Project() ports.ProjectsRepository {
	return r.project
}
