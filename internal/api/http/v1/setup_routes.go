package http

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	docs "github.com/endge-lab/service-backend/internal/api/http/v1/docs"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	health "github.com/endge-lab/service-backend/internal/api/http/v1/health"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	session "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Handler struct {
	SessionHandler   session.SHandler
	ProjectHandler   project.PHandler
	FolderHandler    folder.FHandler
	ComponentHandler component.CHandler
	ConverterHandler converter.ConvHandler
}

func NewHandler(
	sessionHandler session.SHandler,
	projectHandler project.PHandler,
	folderHandler folder.FHandler,
	componentHandler component.CHandler,
	converterHandler converter.ConvHandler,
) *Handler {
	return &Handler{
		SessionHandler:   sessionHandler,
		ProjectHandler:   projectHandler,
		FolderHandler:    folderHandler,
		ComponentHandler: componentHandler,
		ConverterHandler: converterHandler,
	}
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
		docs.RegisterRoutes(app)
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
	project.RegisterRoutes(api, handler.ProjectHandler)
	folder.RegisterRoutes(api, handler.FolderHandler)
	component.RegisterRoutes(api, handler.ComponentHandler)
	converter.RegisterRoutes(api, handler.ConverterHandler)
	app.Use(func(c *fiber.Ctx) error {
		return transport.WriteErrorResponse(c, transport.ErrRouteNotFound)
	})
}
