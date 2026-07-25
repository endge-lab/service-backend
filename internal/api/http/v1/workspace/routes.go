package http

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type WHandler interface {
	Create(*fiber.Ctx) error
	List(*fiber.Ctx) error
	Get(*fiber.Ctx) error
	Update(*fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, h WHandler) {
	r := api.Group("/v1/workspaces")
	r.Get("/", middleware.TraceMiddleware("handler.workspace.list"), h.List)
	r.Post("/", middleware.TraceMiddleware("handler.workspace.create"), h.Create)
	r.Get("/:workspace_identity", middleware.TraceMiddleware("handler.workspace.get"), h.Get)
	r.Patch("/:workspace_identity", middleware.TraceMiddleware("handler.workspace.update"), h.Update)
}
