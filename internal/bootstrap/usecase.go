package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/services"
	"github.com/endge-lab/service-backend/internal/usecase"

	"go.uber.org/fx"
)

func UseCaseModules() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(services.NewTodoFactory, fx.As(new(services.TodoFactory))),
			usecase.NewUseCaseMetrics,
			usecase.NewService,
		),
	)
}
