package middleware

import (
	"sync"

	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var routeTrace struct {
	sync.RWMutex
	tracer trace.Tracer
	logger *zap.Logger
}

// ConfigureTraceMiddleware supplies the shared HTTP tracing dependencies during bootstrap.
func ConfigureTraceMiddleware(tracer trace.Tracer, logger *zap.Logger) {
	routeTrace.Lock()
	defer routeTrace.Unlock()
	routeTrace.tracer = tracer
	routeTrace.logger = logger
}

// TraceMiddleware returns a route-level Fiber handler that traces all subsequent handlers.
func TraceMiddleware(spanName string) fiber.Handler {
	routeTrace.RLock()
	tracer, logger := routeTrace.tracer, routeTrace.logger
	routeTrace.RUnlock()
	if logger == nil {
		logger = zap.NewNop()
	}
	return servicefiber.TraceMiddleware(tracer, logger, spanName)
}
