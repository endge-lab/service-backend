package ai_assistant

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	root := router.Group("/ai")
	root.Get("/capabilities", handler.Capabilities)
	conversations := root.Group("/conversations")
	conversations.Get("/", handler.ListConversations)
	conversations.Post("/", handler.CreateConversation)
	conversations.Post("/reset", handler.ResetConversation)
	conversations.Patch("/:id", handler.PatchConversation)
	conversations.Get("/:id/messages", handler.ListMessages)
	conversations.Post("/:id/runs", handler.Run)
}
