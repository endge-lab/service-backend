package navigation

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/navigations")
	resource.Post("/", handler.Create)
	resource.Get("/", handler.List)
	resource.Get("/:identity", handler.Get)
	resource.Patch("/:identity", handler.Patch)
	resource.Delete("/:identity", handler.Delete)
	resource.Post("/:identity/restore", handler.Restore)
}
