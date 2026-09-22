package commit

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Post("/commits/plan", handler.Plan)
	router.Post("/commits", handler.Create)
	router.Get("/commits", handler.List)
	router.Get("/commits/:id", handler.Get)
	router.Get("/commits/:id/diff", handler.GetDiff)
	router.Post("/commits/:id/restore/plan", handler.PlanRestore)
	router.Post("/commits/:id/restore", handler.Restore)
}
