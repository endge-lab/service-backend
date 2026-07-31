package domain

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type DHandler interface{ DomainHandler }

type DomainHandler interface {
	ListUsages(*fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, handler DHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	routes := api.Group("/v1/domain")
	routes.Use(workspaceMiddleware.RequireWorkspace())
	routes.Get("/usages", middleware.TraceMiddleware("handler.domain.list_usages"), handler.ListUsages)
}
