package folder

import "github.com/gofiber/fiber/v2"

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

	TraceMiddleware(spanName string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, handler FHandler) {
	folders := api.Group("/v1/projects/:project_identity/folders")

	folders.Post("/", handler.TraceMiddleware("handler.folder.create"), handler.CreateFolder)
	folders.Get("/", handler.TraceMiddleware("handler.folder.list"), handler.ListFolders)
	folders.Get("/:folder_identity", handler.TraceMiddleware("handler.folder.get_by_identity"), handler.GetFolderByIdentity)
	folders.Patch("/:folder_identity", handler.TraceMiddleware("handler.folder.update"), handler.UpdateFolder)
	folders.Delete("/:folder_identity", handler.TraceMiddleware("handler.folder.soft_delete"), handler.SoftDeleteFolder)
	folders.Post("/:folder_identity/restore", handler.TraceMiddleware("handler.folder.restore"), handler.RestoreFolder)
	folders.Delete("/:folder_identity/hard", handler.TraceMiddleware("handler.folder.hard_delete"), handler.HardDeleteFolder)
}
