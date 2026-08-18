package domain

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Get("/domain", handler.Live)
	router.Get("/domain/status", handler.Status)
	router.Get("/domain/export", handler.Export)
	router.Post("/domain/import/plan", handler.PlanImport)
	router.Post("/domain/import", handler.Import)
}
