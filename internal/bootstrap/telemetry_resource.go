package bootstrap

import (
	"context"

	"github.com/endge-lab/service-template-go/internal/config"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func newTelemetryResource(cfg *config.Config) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.AppName),
			semconv.ServiceVersion(cfg.AppVersion),
			attribute.String("deployment.environment", cfg.AppEnv),
		),
	)
}
