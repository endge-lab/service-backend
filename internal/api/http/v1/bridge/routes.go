package bridge

import "github.com/gofiber/fiber/v2"

// RegisterPublicRoutes registers the dev client endpoint before auth middleware.
func RegisterPublicRoutes(router fiber.Router, handler *Handler) {
	router.Get("/api/v1/bridge/client", handler.Client)
}

// RegisterRoutes registers the configurator endpoint on the authenticated v1 group.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Get("/bridge/configurator", handler.Configurator)
}
