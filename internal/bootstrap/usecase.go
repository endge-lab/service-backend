package bootstrap

import (
	"github.com/endge-lab/service-template-go/internal/api/http"
	"github.com/endge-lab/service-template-go/internal/auth"
	"github.com/endge-lab/service-template-go/internal/middleware"
	"github.com/endge-lab/service-template-go/internal/ports"
	"github.com/endge-lab/service-template-go/internal/repo/postgres"
	"github.com/endge-lab/service-template-go/internal/services"
	"github.com/endge-lab/service-template-go/internal/usecase"

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
			fx.Annotate(usecase.NewLoadSessionUseCase, fx.As(new(usecase.LoadSessionUseCase))),
			fx.Annotate(usecase.NewCreateTodoUseCase, fx.As(new(usecase.CreateTodoUseCase))),
			http.NewHandler,
		),
	)
}
