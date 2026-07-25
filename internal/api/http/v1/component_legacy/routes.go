package component_legacy

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

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
}

func RegisterRoutes(api fiber.Router, h ComponentLegacyHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	r := api.Group("/v1/projects/:project_identity/components-legacy")
	r.Use(workspaceMiddleware.RequireWorkspace())
	r.Post("/", middleware.TraceMiddleware("handler.component_legacy.create"), h.Create)
	r.Get("/", middleware.TraceMiddleware("handler.component_legacy.list"), h.List)
	r.Get("/:component_identity", middleware.TraceMiddleware("handler.component_legacy.get"), h.GetByIdentity)
	r.Patch("/:component_identity", middleware.TraceMiddleware("handler.component_legacy.update"), h.Update)
	r.Delete("/:component_identity", middleware.TraceMiddleware("handler.component_legacy.delete"), h.SoftDelete)
	r.Post("/:component_identity/restore", middleware.TraceMiddleware("handler.component_legacy.restore"), h.Restore)
	r.Delete("/:component_identity/hard", middleware.TraceMiddleware("handler.component_legacy.hard_delete"), h.HardDelete)
}
