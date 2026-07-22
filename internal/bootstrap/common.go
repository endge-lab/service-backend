package bootstrap

import (
	"context"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/platform"
	servicetelemetry "github.com/endge-lab/service-kit-go/pkg/telemetry"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func CommonModules() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			observability.NewCore,
			newPostgres,
			newOpenSearchLogExporter,
			InitLogger,
			InitValidator,
			platform.NewRedpandaClient,
			NewFiber,
			newTelemetryProviders,
			newTracer,
			newMeter,
		),
		fx.Invoke(registerObservabilityShutdown, registerPrometheusMetricsServer),
	)
}

func registerObservabilityShutdown(
	lc fx.Lifecycle,
	logger *zap.Logger,
	openSearch *openSearchLogExporter,
	providers *servicetelemetry.Providers,
) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down observability")
			_ = logger.Sync()

			if openSearch != nil && openSearch.exporter != nil {
				if err := openSearch.exporter.Shutdown(ctx); err != nil {
					return err
				}
			}

			return providers.Shutdown(ctx)
		},
	})
}
