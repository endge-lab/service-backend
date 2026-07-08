package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/usecase"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.uber.org/fx"
)

func UseCaseModules() fx.Option {
	return fx.Options(
		fx.Provide(
			shared.NewUseCaseMetrics,
			usecase.NewService,
		),
	)
}
