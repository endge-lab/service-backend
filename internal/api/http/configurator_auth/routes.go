package configurator_auth

import "github.com/gofiber/fiber/v2"

func RegisterPublicRoutes(app fiber.Router, handler *Handler) {
	group := app.Group("/auth")
	group.Get("/login", handler.Login)
	group.Get("/callback", handler.Callback)
	group.Post("/logout", handler.Logout)
}
