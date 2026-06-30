package middleware

import (
	servicefiber "github.com/endge-lab/service-kit-go/httpkit/fiber"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func RequestLogger(log *zap.Logger) fiber.Handler {
	return servicefiber.RequestLoggerMiddleware(log)
}
