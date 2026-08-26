package ai_catalog

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	root := router.Group("/ai")
	root.Get("/provider-adapters", handler.Adapters)
	connections := root.Group("/provider-connections")
	connections.Get("/", handler.ListConnections)
	connections.Post("/", handler.CreateConnection)
	connections.Patch("/:id", handler.PatchConnection)
	connections.Put("/:id/credential", handler.ReplaceCredential)
	connections.Delete("/:id", handler.DeleteConnection)
	models := root.Group("/model-profiles")
	models.Get("/", handler.ListModels)
	models.Post("/", handler.CreateModel)
	models.Patch("/:id", handler.PatchModel)
	models.Delete("/:id", handler.DeleteModel)
}
