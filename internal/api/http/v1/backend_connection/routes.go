package backend_connection

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/backend-connections")
	resource.Get("/", handler.List)
	resource.Post("/", handler.Create)
	resource.Delete("/:id", handler.Delete)
}
