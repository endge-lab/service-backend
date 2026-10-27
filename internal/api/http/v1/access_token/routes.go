package access_token

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует управление персональными токенами вне workspace scope.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Post("/access-tokens", handler.Create)
	router.Get("/access-tokens", handler.List)
	router.Delete("/access-tokens/:id", handler.Revoke)
}
