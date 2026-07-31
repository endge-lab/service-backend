package bootstrap

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	servicetelemetry "github.com/endge-lab/service-kit-go/pkg/telemetry"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	traceSampleModeAlways = "always"
	traceSampleModeNever  = "never"
)

func newTelemetryProviders(cfg *config.Config, logger *zap.Logger) (*servicetelemetry.Providers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpoint := ""
	traceSampleMode := traceSampleModeNever
	if cfg.Telemetry.Enabled {
		endpoint = cfg.Telemetry.OTLPEndpoint
		traceSampleMode = traceSampleModeAlways
	}

	providers, err := servicetelemetry.NewProviders(ctx, servicetelemetry.Config{
		ServiceName:       cfg.App.Name,
		ServiceVersion:    cfg.App.Version,
		Environment:       cfg.App.Env,
		OTLPEnabled:       cfg.Telemetry.Enabled,
		OTLPEndpoint:      endpoint,
		OTLPInsecure:      cfg.Telemetry.OTLPInsecure,
		MetricsInterval:   15 * time.Second,
		PrometheusEnabled: cfg.Metrics.Enabled,
		TraceSampleMode:   traceSampleMode,
	}, logger)
	if err != nil {
		logger.Warn("telemetry exporter disabled", zap.Error(err), zap.String("endpoint", endpoint))
		providers, err = servicetelemetry.NewProviders(ctx, servicetelemetry.Config{
			ServiceName:       cfg.App.Name,
			ServiceVersion:    cfg.App.Version,
			Environment:       cfg.App.Env,
			OTLPEnabled:       false,
			PrometheusEnabled: cfg.Metrics.Enabled,
			TraceSampleMode:   traceSampleModeNever,
		}, logger)
		if err != nil {
			return nil, err
		}
	}

	return providers, nil
}

func newTracer(cfg *config.Config, providers *servicetelemetry.Providers) trace.Tracer {
	return providers.Tracer(cfg.App.Name)
}

func newMeter(cfg *config.Config, providers *servicetelemetry.Providers) metric.Meter {
	return providers.Meter(cfg.App.Name)
}
