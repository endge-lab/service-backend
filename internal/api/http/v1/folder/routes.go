package folder

import (
	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

type FHandler interface {
	FolderHandler
}

type FolderHandler interface {
	CreateFolder(c *fiber.Ctx) error
	GetFolderByIdentity(c *fiber.Ctx) error
	ListFolders(c *fiber.Ctx) error
	UpdateFolder(c *fiber.Ctx) error
	SoftDeleteFolder(c *fiber.Ctx) error
	RestoreFolder(c *fiber.Ctx) error
	HardDeleteFolder(c *fiber.Ctx) error
}

func RegisterRoutes(api fiber.Router, handler FHandler, workspaceMiddleware *middleware.WorkspaceContextMiddleware) {
	folders := api.Group("/v1/projects/:project_identity/folders")
	folders.Use(workspaceMiddleware.RequireWorkspace())

	folders.Post("/", middleware.TraceMiddleware("handler.folder.create"), handler.CreateFolder)
	folders.Get("/", middleware.TraceMiddleware("handler.folder.list"), handler.ListFolders)
	folders.Get("/:folder_identity", middleware.TraceMiddleware("handler.folder.get_by_identity"), handler.GetFolderByIdentity)
	folders.Patch("/:folder_identity", middleware.TraceMiddleware("handler.folder.update"), handler.UpdateFolder)
	folders.Delete("/:folder_identity", middleware.TraceMiddleware("handler.folder.soft_delete"), handler.SoftDeleteFolder)
	folders.Post("/:folder_identity/restore", middleware.TraceMiddleware("handler.folder.restore"), handler.RestoreFolder)
	folders.Delete("/:folder_identity/hard", middleware.TraceMiddleware("handler.folder.hard_delete"), handler.HardDeleteFolder)
}
