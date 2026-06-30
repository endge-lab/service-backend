package middleware

import (
	servicefiber "github.com/endge-lab/service-kit-go/httpkit/fiber"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TraceMiddleware(tracer trace.Tracer, logger *zap.Logger, name string, attrs ...attribute.KeyValue) fiber.Handler {
	return servicefiber.TraceMiddleware(tracer, logger, name, attrs...)
}
