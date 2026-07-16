package http

import (
	"github.com/endge-lab/service-backend/internal/api/http/health"
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/openapi"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/session"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component_legacy"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	"github.com/endge-lab/service-backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Handler struct {
	SessionHandler         session.SHandler
	ProjectHandler         project.PHandler
	FolderHandler          folder.FHandler
	ComponentLegacyHandler component_legacy.CHandler
	ConverterHandler       converter.ConvHandler
	QueryHandler           query.QHandler
	DataViewHandler        data_view.DVHandler
}

func NewHandler(
	sessionHandler session.SHandler,
	projectHandler project.PHandler,
	folderHandler folder.FHandler,
	componentLegacyHandler component_legacy.CHandler,
	converterHandler converter.ConvHandler,
	queryHandler query.QHandler,
	dataViewHandler data_view.DVHandler,
) *Handler {
	return &Handler{
		SessionHandler:         sessionHandler,
		ProjectHandler:         projectHandler,
		FolderHandler:          folderHandler,
		ComponentLegacyHandler: componentLegacyHandler,
		ConverterHandler:       converterHandler,
		QueryHandler:           queryHandler,
		DataViewHandler:        dataViewHandler,
	}
}

func SetupRoutes(
	app *fiber.App,
	cfg *config.Config,
	handler *Handler,
	authMiddleware httpmiddleware.AuthMiddleware,
	meter metric.Meter,
	logger *zap.Logger,
) {
	httpmiddleware.Register(app, cfg, meter, logger)

	if !cfg.App.IsProduction() {
		openapi.RegisterRoutes(app)
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
	component_legacy.RegisterRoutes(api, handler.ComponentLegacyHandler)
	converter.RegisterRoutes(api, handler.ConverterHandler)
	query.RegisterRoutes(api, handler.QueryHandler)
	data_view.RegisterRoutes(api, handler.DataViewHandler)
	app.Use(func(c *fiber.Ctx) error {
		return respond.WriteErrorResponse(c, respond.ErrRouteNotFound)
	})
}
