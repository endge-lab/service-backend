package http

import "github.com/gofiber/fiber/v2"

type THandler interface {
	TodoHandler
}

type TodoHandler interface {
	CreateTodo(c *fiber.Ctx) error
	TraceMiddleware(spanName string) fiber.Handler
}

func RegisterRoutes(api fiber.Router, handler THandler) {
	api.Post("/todos", handler.TraceMiddleware("handler.create_todo"), handler.CreateTodo)
}
