package access_control

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Get("/service-users/search", handler.SearchUsers)
	router.Get("/access-grants", handler.List)
	router.Put("/access-grants", handler.Put)
	router.Delete("/access-grants/:id", handler.Delete)
	router.Post("/access-grants/bulk-workspaces", handler.Bulk)
}
