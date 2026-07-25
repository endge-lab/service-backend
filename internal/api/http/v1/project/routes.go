package project

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type PHandler interface {
	ProjectHandler
}

type ProjectHandler interface {
	CreateProject(c *fiber.Ctx) error
	GetProjectByIdentity(c *fiber.Ctx) error
	ListProjects(c *fiber.Ctx) error
	UpdateProject(c *fiber.Ctx) error
	SoftDeleteProject(c *fiber.Ctx) error
	RestoreProject(c *fiber.Ctx) error
	HardDeleteProject(c *fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, handler PHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	projects := api.Group("/v1/projects")
	projects.Use(workspaceMiddleware.RequireWorkspace())

	projects.Post("/", middleware.TraceMiddleware("handler.project.create"), handler.CreateProject)
	projects.Get("/", middleware.TraceMiddleware("handler.project.list"), handler.ListProjects)
	projects.Get("/:project_identity", middleware.TraceMiddleware("handler.project.get_by_identity"), handler.GetProjectByIdentity)
	projects.Patch("/:project_identity", middleware.TraceMiddleware("handler.project.update"), handler.UpdateProject)
	projects.Delete("/:project_identity", middleware.TraceMiddleware("handler.project.soft_delete"), handler.SoftDeleteProject)
	projects.Post("/:project_identity/restore", middleware.TraceMiddleware("handler.project.restore"), handler.RestoreProject)
	projects.Delete("/:project_identity/hard", middleware.TraceMiddleware("handler.project.hard_delete"), handler.HardDeleteProject)
}
