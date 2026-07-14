package converter

import "github.com/gofiber/fiber/v2"

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
	TraceMiddleware(string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, h ConverterHandler) {
	r := api.Group("/v1/projects/:project_identity/converters")
	r.Post("/", h.TraceMiddleware("handler.converter.create"), h.Create)
	r.Get("/", h.TraceMiddleware("handler.converter.list"), h.List)
	r.Get("/:converter_identity", h.TraceMiddleware("handler.converter.get"), h.GetByIdentity)
	r.Patch("/:converter_identity", h.TraceMiddleware("handler.converter.update"), h.Update)
	r.Delete("/:converter_identity", h.TraceMiddleware("handler.converter.delete"), h.SoftDelete)
	r.Post("/:converter_identity/restore", h.TraceMiddleware("handler.converter.restore"), h.Restore)
	r.Delete("/:converter_identity/hard", h.TraceMiddleware("handler.converter.hard_delete"), h.HardDelete)
}
