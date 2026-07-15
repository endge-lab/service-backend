package project

import "github.com/gofiber/fiber/v2"

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

	TraceMiddleware(spanName string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, handler PHandler) {
	projects := api.Group("/v1/projects")

	projects.Post("/", handler.TraceMiddleware("handler.project.create"), handler.CreateProject)
	projects.Get("/", handler.TraceMiddleware("handler.project.list"), handler.ListProjects)
	projects.Get("/:project_identity", handler.TraceMiddleware("handler.project.get_by_identity"), handler.GetProjectByIdentity)
	projects.Patch("/:project_identity", handler.TraceMiddleware("handler.project.update"), handler.UpdateProject)
	projects.Delete("/:project_identity", handler.TraceMiddleware("handler.project.soft_delete"), handler.SoftDeleteProject)
	projects.Post("/:project_identity/restore", handler.TraceMiddleware("handler.project.restore"), handler.RestoreProject)
	projects.Delete("/:project_identity/hard", handler.TraceMiddleware("handler.project.hard_delete"), handler.HardDeleteProject)
}
