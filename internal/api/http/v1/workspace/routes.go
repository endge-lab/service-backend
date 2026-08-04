package workspace

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует HTTP-маршруты ресурса.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	router.Get("/workspaces", handler.List)
	router.Post("/workspaces", handler.Create)
	router.Get("/workspaces/:identity", handler.Get)
	router.Patch("/workspaces/:identity", handler.Patch)
	router.Get("/workspaces/:identity/members", handler.ListMembers)
	router.Put("/workspaces/:identity/members/:userId", handler.PutMember)
	router.Delete("/workspaces/:identity/members/:userId", handler.DeleteMember)
}
