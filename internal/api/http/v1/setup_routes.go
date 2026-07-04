package http

import (
	docs "github.com/endge-lab/service-backend/internal/api/http/v1/docs"
	health "github.com/endge-lab/service-backend/internal/api/http/v1/health"
	session "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	todo "github.com/endge-lab/service-backend/internal/api/http/v1/todo"
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Handler struct {
	SessionHandler session.SHandler
	TodoHandler    todo.THandler
}

func NewHandler(sessionHandler session.SHandler, todoHandler todo.THandler) *Handler {
	return &Handler{SessionHandler: sessionHandler, TodoHandler: todoHandler}
}

func SetupRoutes(
	app *fiber.App,
	cfg *config.Config,
	handler *Handler,
	authMiddleware middleware.AuthMiddleware,
	meter metric.Meter,
	logger *zap.Logger,
) {
	setupMiddlewares(app, cfg, meter, logger)

	if !cfg.App.IsProduction() {
		docs.RegisterRoutes(app, docs.Config{
			OpenAPIPath: "./docs/openapi.yaml",
		})
	}

	health.RegisterRoutes(app, health.Config{
		Service: cfg.App.Name,
		Version: cfg.App.Version,
		Env:     cfg.App.Env,
	})

	api := app.Group("/api")
	if cfg.Auth.Enabled {
		api.Use(authMiddleware.AuthMiddleware())
		session.RegisterRoutes(api, handler.SessionHandler)
	}
	todo.RegisterRoutes(api, handler.TodoHandler)

	app.Use(func(c *fiber.Ctx) error {
		return transport.WriteErrorResponse(c, transport.ErrRouteNotFound)
	})
}
