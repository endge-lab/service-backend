package component_legacy

import "github.com/gofiber/fiber/v2"

type CHandler interface {
	ComponentLegacyHandler
}

type ComponentLegacyHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
	TraceMiddleware(string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, h ComponentLegacyHandler) {
	r := api.Group("/v1/projects/:project_identity/components-legacy")
	r.Post("/", h.TraceMiddleware("handler.component_legacy.create"), h.Create)
	r.Get("/", h.TraceMiddleware("handler.component_legacy.list"), h.List)
	r.Get("/:component_identity", h.TraceMiddleware("handler.component_legacy.get"), h.GetByIdentity)
	r.Patch("/:component_identity", h.TraceMiddleware("handler.component_legacy.update"), h.Update)
	r.Delete("/:component_identity", h.TraceMiddleware("handler.component_legacy.delete"), h.SoftDelete)
	r.Post("/:component_identity/restore", h.TraceMiddleware("handler.component_legacy.restore"), h.Restore)
	r.Delete("/:component_identity/hard", h.TraceMiddleware("handler.component_legacy.hard_delete"), h.HardDelete)
}
