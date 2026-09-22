package workspace

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Get("/workspaces", handler.List)
	router.Post("/workspaces", handler.Create)
	router.Get("/workspaces/archive", handler.ListArchive)
	router.Post("/workspaces/:identity/restore", handler.Restore)
	router.Get("/workspaces/:identity", handler.Get)
	router.Patch("/workspaces/:identity", handler.Patch)
	router.Delete("/workspaces/:identity", handler.Delete)
	router.Get("/workspaces/:identity/members", handler.ListMembers)
	router.Put("/workspaces/:identity/members/:userId", handler.PutMember)
	router.Delete("/workspaces/:identity/members/:userId", handler.DeleteMember)
}
