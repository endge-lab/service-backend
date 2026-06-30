package bootstrap

import (
	"github.com/endge-lab/service-template-go/internal/api/http"

	"go.uber.org/fx"
)

func InvokeModules() fx.Option {
	return fx.Options(
		fx.Invoke(
			Migrate,
			http.SetupRoutes,
		),
	)
}
