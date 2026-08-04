package backup

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует read/write HTTP-маршруты snapshot backups.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/domain/backups")
	resource.Post("/", handler.Create)
	resource.Get("/", handler.List)
	resource.Get("/archive", handler.Archive)
	resource.Get("/:id/export", handler.Export)
	resource.Get("/:id", handler.Get)
}
