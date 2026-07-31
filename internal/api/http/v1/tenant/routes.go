package tenant

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type THandler interface{ TenantHandler }

type TenantHandler interface {
	CreateTenant(*fiber.Ctx) error
	ListTenants(*fiber.Ctx) error
	GetTenantByIdentity(*fiber.Ctx) error
	UpdateTenant(*fiber.Ctx) error
	HardDeleteTenant(*fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, handler THandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	r := api.Group("/v1/tenants")
	r.Use(workspaceMiddleware.RequireWorkspace())
	r.Post("/", middleware.TraceMiddleware("handler.tenant.create"), handler.CreateTenant)
	r.Get("/", middleware.TraceMiddleware("handler.tenant.list"), handler.ListTenants)
	r.Get("/:tenant_identity", middleware.TraceMiddleware("handler.tenant.get_by_identity"), handler.GetTenantByIdentity)
	r.Patch("/:tenant_identity", middleware.TraceMiddleware("handler.tenant.update"), handler.UpdateTenant)
	r.Delete("/:tenant_identity", middleware.TraceMiddleware("handler.tenant.hard_delete"), handler.HardDeleteTenant)
}
