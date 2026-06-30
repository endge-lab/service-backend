package bootstrap

import (
	"github.com/endge-lab/service-template-go/internal/config"
	"github.com/endge-lab/service-template-go/internal/platform"

	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"
)

func CommonModules() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			newPostgres,
			InitLogger,
			InitValidator,
			platform.NewRedpandaClient,
			NewFiber,
			newTelemetryResource,
			newTextMapPropagator,
			newTraceProvider,
			newMeterProvider,
			newTracer,
			newMeter,
		),
		fx.Invoke(
			func(propagator propagation.TextMapPropagator) {
				registerTextMapPropagator(propagator)
			},
		),
	)
}
