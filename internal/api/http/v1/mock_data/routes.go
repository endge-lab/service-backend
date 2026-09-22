package mock_data

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, h *Handler) {
	r := router.Group("/mock-data")
	r.Get("/capabilities", h.Capabilities)
	r.Post("/generate", h.Generate)
	r.Post("/streams", h.Create)
	r.Get("/streams/:id", h.Get)
	r.Get("/streams/:id/events", h.Events)
	r.Patch("/streams/:id", h.Update)
	r.Post("/streams/:id/keepalive", h.KeepAlive)
	r.Delete("/streams/:id", h.Stop)
}
