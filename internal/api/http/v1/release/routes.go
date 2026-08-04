package release

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/releases")
	resource.Post("/", handler.Create)
	resource.Get("/", handler.List)
	resource.Get("/:identity", handler.Get)
	resource.Get("/:identity/export", handler.Export)
	resource.Post("/:identity/restore/plan", handler.PlanRestore)
	resource.Post("/:identity/restore", handler.Restore)
}
