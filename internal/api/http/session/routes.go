// Package session implements the unversioned HTTP session adapter.
package session

import "github.com/gofiber/fiber/v2"

type SHandler interface {
	SessionHandler
}

type SessionHandler interface {
	LoadSession(c *fiber.Ctx) error
	TraceMiddleware(spanName string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, handler SHandler) {
	api.Get("/session/me", handler.TraceMiddleware("handler.load_session"), handler.LoadSession)
}
