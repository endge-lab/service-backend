package middleware

import (
	servicefiber "github.com/endge-lab/service-kit-go/httpkit/fiber"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric"
)

func NewRequestMetricsMiddleware(meter metric.Meter) (fiber.Handler, error) {
	return servicefiber.NewRequestMetricsMiddleware(meter)
}
