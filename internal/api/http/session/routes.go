// Package session implements the unversioned HTTP session adapter.
package session

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type SHandler interface {
	SessionHandler
}

type SessionHandler interface {
	LoadSession(c *fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, handler SHandler) {
	api.Get("/session/me", middleware.TraceMiddleware("handler.load_session"), handler.LoadSession)
}
