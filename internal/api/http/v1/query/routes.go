package query

import "github.com/gofiber/fiber/v2"

type QHandler interface{ QueryHandler }

type QueryHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
	TraceMiddleware(string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, h QueryHandler) {
	r := api.Group("/v1/projects/:project_identity/queries")
	r.Post("/", h.TraceMiddleware("handler.query.create"), h.Create)
	r.Get("/", h.TraceMiddleware("handler.query.list"), h.List)
	r.Get("/:query_identity", h.TraceMiddleware("handler.query.get"), h.GetByIdentity)
	r.Patch("/:query_identity", h.TraceMiddleware("handler.query.update"), h.Update)
	r.Delete("/:query_identity", h.TraceMiddleware("handler.query.delete"), h.SoftDelete)
	r.Post("/:query_identity/restore", h.TraceMiddleware("handler.query.restore"), h.Restore)
	r.Delete("/:query_identity/hard", h.TraceMiddleware("handler.query.hard_delete"), h.HardDelete)
}
