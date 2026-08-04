package revision

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/domain/documents/:type/:identity/revisions")
	resource.Get("/", handler.List)
	resource.Get("/:revisionId", handler.Get)
	resource.Post("/:revisionId/restore", handler.Restore)
}
