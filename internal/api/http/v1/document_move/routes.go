package document_move

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует атомарные операции над несколькими domain-документами.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Post("/domain/documents/move", handler.Move)
}
