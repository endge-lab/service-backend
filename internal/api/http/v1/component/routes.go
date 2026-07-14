package component

import "github.com/gofiber/fiber/v2"

type CHandler interface {
	ComponentHandler
}

type ComponentHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
	TraceMiddleware(string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, h ComponentHandler) {
	r := api.Group("/v1/projects/:project_identity/components")
	r.Post("/", h.TraceMiddleware("handler.component.create"), h.Create)
	r.Get("/", h.TraceMiddleware("handler.component.list"), h.List)
	r.Get("/:component_identity", h.TraceMiddleware("handler.component.get"), h.GetByIdentity)
	r.Patch("/:component_identity", h.TraceMiddleware("handler.component.update"), h.Update)
	r.Delete("/:component_identity", h.TraceMiddleware("handler.component.delete"), h.SoftDelete)
	r.Post("/:component_identity/restore", h.TraceMiddleware("handler.component.restore"), h.Restore)
	r.Delete("/:component_identity/hard", h.TraceMiddleware("handler.component.hard_delete"), h.HardDelete)
}
