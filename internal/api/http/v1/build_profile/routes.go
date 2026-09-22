package build_profile

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	root := router.Group("/build-profiles")
	root.Get("/", handler.List)
	root.Post("/", handler.Create)
	root.Patch("/:identity", handler.Patch)
	root.Delete("/:identity", handler.Delete)
}
