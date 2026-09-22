package configuration

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/configurations")
	resource.Post("/", handler.Create)
	resource.Get("/", handler.List)
	resource.Get("/:identity", handler.Get)
	resource.Patch("/:identity", handler.Patch)
	resource.Delete("/:identity", handler.Delete)
	resource.Post("/:identity/restore", handler.Restore)
}
