package bootstrap

import (
	"context"

	"github.com/endge-lab/service-backend/internal/config"
	servicetelemetry "github.com/endge-lab/service-kit-go/pkg/telemetry"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func registerPrometheusMetricsServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	providers *servicetelemetry.Providers,
	logger *zap.Logger,
) error {
	if !cfg.Metrics.Enabled {
		return nil
	}

	server, err := servicetelemetry.NewPrometheusServer(
		servicetelemetry.PrometheusServerConfig{
			BindAddress: cfg.Metrics.BindAddress,
			HandlerPath: cfg.Metrics.HandlerPath,
		},
		providers.PrometheusHandler(),
		logger,
	)
	if err != nil {
		return err
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return server.Start()
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return nil
}
