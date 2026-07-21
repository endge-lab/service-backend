package converter

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type ConvHandler interface {
	ConverterHandler
}

type ConverterHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, h ConverterHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	r := api.Group("/v1/projects/:project_identity/converters")
	r.Use(workspaceMiddleware.RequireWorkspace())
	r.Post("/", middleware.TraceMiddleware("handler.converter.create"), h.Create)
	r.Get("/", middleware.TraceMiddleware("handler.converter.list"), h.List)
	r.Get("/:converter_identity", middleware.TraceMiddleware("handler.converter.get"), h.GetByIdentity)
	r.Patch("/:converter_identity", middleware.TraceMiddleware("handler.converter.update"), h.Update)
	r.Delete("/:converter_identity", middleware.TraceMiddleware("handler.converter.delete"), h.SoftDelete)
	r.Post("/:converter_identity/restore", middleware.TraceMiddleware("handler.converter.restore"), h.Restore)
	r.Delete("/:converter_identity/hard", middleware.TraceMiddleware("handler.converter.hard_delete"), h.HardDelete)
}
