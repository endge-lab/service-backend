package integration

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/integrations")
	resource.Get("/", handler.List)
	resource.Post("/", handler.Create)
	resource.Get("/:identity", handler.Get)
	resource.Patch("/:identity", handler.Patch)
	resource.Delete("/:identity", handler.Delete)
	resource.Post("/:identity/restore", handler.Restore)
}
