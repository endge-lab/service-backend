package query

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type QHandler interface{ QueryHandler }

type QueryHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	GetByIdentity(*fiber.Ctx) error
	Update(*fiber.Ctx) error
	SoftDelete(*fiber.Ctx) error
	Restore(*fiber.Ctx) error
	HardDelete(*fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, h QueryHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	r := api.Group("/v1/projects/:project_identity/queries")
	r.Use(workspaceMiddleware.RequireWorkspace())
	r.Post("/", middleware.TraceMiddleware("handler.query.create"), h.Create)
	r.Get("/", middleware.TraceMiddleware("handler.query.list"), h.List)
	r.Get("/:query_identity", middleware.TraceMiddleware("handler.query.get"), h.GetByIdentity)
	r.Patch("/:query_identity", middleware.TraceMiddleware("handler.query.update"), h.Update)
	r.Delete("/:query_identity", middleware.TraceMiddleware("handler.query.delete"), h.SoftDelete)
	r.Post("/:query_identity/restore", middleware.TraceMiddleware("handler.query.restore"), h.Restore)
	r.Delete("/:query_identity/hard", middleware.TraceMiddleware("handler.query.hard_delete"), h.HardDelete)
}
