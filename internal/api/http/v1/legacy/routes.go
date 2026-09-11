package legacy

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует временные миграционные маршруты.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Post("/legacy/workspace-folders/rebuild-from-frontend", handler.RebuildWorkspaceFoldersFromFrontend)
}
