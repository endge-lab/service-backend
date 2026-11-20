package user_state

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, h *Handler) {
	root := router.Group("/user-state")
	root.Get("/", h.self)
	root.Patch("/", h.saveSelf)
	root.Get("/admin/contexts", h.contexts)
	root.Get("/admin/users", h.users)
	root.Get("/admin/users/:user", h.readUser)
	root.Patch("/admin/users/:user", h.saveUser)
	root.Post("/admin/reset", h.reset)
}
