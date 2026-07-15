package data_view

import "github.com/gofiber/fiber/v2"

type DVHandler interface{ DataViewHandler }

type DataViewHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
	TraceMiddleware(string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, h DataViewHandler) {
	r := api.Group("/v1/projects/:project_identity/data-views")
	r.Post("/", h.TraceMiddleware("handler.data_view.create"), h.Create)
	r.Get("/", h.TraceMiddleware("handler.data_view.list"), h.List)
	r.Get("/:data_view_identity", h.TraceMiddleware("handler.data_view.get"), h.GetByIdentity)
	r.Patch("/:data_view_identity", h.TraceMiddleware("handler.data_view.update"), h.Update)
	r.Delete("/:data_view_identity", h.TraceMiddleware("handler.data_view.delete"), h.SoftDelete)
	r.Post("/:data_view_identity/restore", h.TraceMiddleware("handler.data_view.restore"), h.Restore)
	r.Delete("/:data_view_identity/hard", h.TraceMiddleware("handler.data_view.hard_delete"), h.HardDelete)
}
