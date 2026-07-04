package bootstrap

import (
	v1 "github.com/endge-lab/service-backend/internal/api/http/v1"
	session "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	todo "github.com/endge-lab/service-backend/internal/api/http/v1/todo"
	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/internal/middleware"
	"github.com/endge-lab/service-backend/internal/ports"
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	"github.com/endge-lab/service-backend/internal/services"
	"github.com/endge-lab/service-backend/internal/usecase"

	"go.uber.org/fx"
)

func UseCaseModules() fx.Option {
	return fx.Options(
		fx.Provide(
			auth.NewResolver,
			fx.Annotate(middleware.NewAuthMiddleware, fx.As(new(middleware.AuthMiddleware))),
			fx.Annotate(postgres.NewTxManager, fx.As(new(ports.TxManager))),
			fx.Annotate(postgres.NewUserRepository, fx.As(new(ports.UserRepository))),
			fx.Annotate(postgres.NewTodoRepository, fx.As(new(ports.TodoRepository))),
			fx.Annotate(services.NewTodoFactory, fx.As(new(services.TodoFactory))),
			usecase.NewUseCaseMetrics,
			usecase.NewService,
			fx.Annotate(session.NewHandler, fx.As(new(session.SHandler))),
			fx.Annotate(todo.NewHandler, fx.As(new(todo.THandler))),
			v1.NewHandler,
		),
	)
}
