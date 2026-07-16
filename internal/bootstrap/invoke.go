package bootstrap

import (
	httpapi "github.com/endge-lab/service-backend/internal/api/http"

	"go.uber.org/fx"
)

func InvokeModules() fx.Option {
	return fx.Options(
		fx.Invoke(
			httpapi.SetupRoutes,
		),
	)
}
