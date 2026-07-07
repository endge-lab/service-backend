package project

import "github.com/gofiber/fiber/v2"

type PHandler interface {
	ProjectHandler
}

type ProjectHandler interface {
	CreateProject(c *fiber.Ctx) error
	GetProjectByID(c *fiber.Ctx) error
	GetProjectByIdentity(c *fiber.Ctx) error
	ListProjects(c *fiber.Ctx) error
	UpdateProject(c *fiber.Ctx) error
	SoftDeleteProject(c *fiber.Ctx) error
	RestoreProject(c *fiber.Ctx) error
	HardDeleteProject(c *fiber.Ctx) error
	CountProjects(c *fiber.Ctx) error

	TraceMiddleware(spanName string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, handler PHandler) {
	projects := api.Group("/projects")

	projects.Post("/", handler.TraceMiddleware("handler.project.create"), handler.CreateProject)
	projects.Get("/", handler.TraceMiddleware("handler.project.list"), handler.ListProjects)
	projects.Get("/count", handler.TraceMiddleware("handler.project.count"), handler.CountProjects)

	projects.Get("/:id", handler.TraceMiddleware("handler.project.get_by_id"), handler.GetProjectByID)
	projects.Get("/identity/:identity", handler.TraceMiddleware("handler.project.get_by_identity"), handler.GetProjectByIdentity)

	projects.Patch("/:id", handler.TraceMiddleware("handler.project.update"), handler.UpdateProject)
	projects.Delete("/:id", handler.TraceMiddleware("handler.project.soft_delete"), handler.SoftDeleteProject)

	projects.Post("/:id/restore", handler.TraceMiddleware("handler.project.restore"), handler.RestoreProject)
	projects.Delete("/:id/hard", handler.TraceMiddleware("handler.project.hard_delete"), handler.HardDeleteProject)
}
