package http

import (
	configuratorauth "github.com/endge-lab/service-backend/internal/api/http/configurator_auth"
	"github.com/endge-lab/service-backend/internal/api/http/health"
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/openapi"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/action"
	"github.com/endge-lab/service-backend/internal/api/http/v1/auth_profile"
	"github.com/endge-lab/service-backend/internal/api/http/v1/backup"
	"github.com/endge-lab/service-backend/internal/api/http/v1/commit"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/composition"
	"github.com/endge-lab/service-backend/internal/api/http/v1/computation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/domain"
	"github.com/endge-lab/service-backend/internal/api/http/v1/environment"
	"github.com/endge-lab/service-backend/internal/api/http/v1/filter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/i18n_bundle"
	"github.com/endge-lab/service-backend/internal/api/http/v1/integration"
	"github.com/endge-lab/service-backend/internal/api/http/v1/mock"
	"github.com/endge-lab/service-backend/internal/api/http/v1/navigation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	"github.com/endge-lab/service-backend/internal/api/http/v1/release"
	"github.com/endge-lab/service-backend/internal/api/http/v1/revision"
	httpsession "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	"github.com/endge-lab/service-backend/internal/api/http/v1/store"
	"github.com/endge-lab/service-backend/internal/api/http/v1/stream"
	"github.com/endge-lab/service-backend/internal/api/http/v1/style"
	"github.com/endge-lab/service-backend/internal/api/http/v1/tenant"
	domain_type "github.com/endge-lab/service-backend/internal/api/http/v1/type"
	"github.com/endge-lab/service-backend/internal/api/http/v1/update"
	"github.com/endge-lab/service-backend/internal/api/http/v1/vocab"
	"github.com/endge-lab/service-backend/internal/api/http/v1/workspace"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handlers struct {
	fx.In

	CurrentUser      *httpmiddleware.CurrentUserMiddleware
	ConfiguratorAuth *configuratorauth.Handler
	Workspace        *workspace.Handler
	Session          *httpsession.Handler
	Integration      *integration.Handler
	Project          *project.Handler
	Tenant           *tenant.Handler
	Environment      *environment.Handler
	Folder           *folder.Handler
	Type             *domain_type.Handler
	Query            *query.Handler
	DataView         *data_view.Handler
	Composition      *composition.Handler
	Store            *store.Handler
	Stream           *stream.Handler
	Update           *update.Handler
	Mock             *mock.Handler
	Component        *component.Handler
	Action           *action.Handler
	Filter           *filter.Handler
	Converter        *converter.Handler
	Computation      *computation.Handler
	Vocab            *vocab.Handler
	I18nBundle       *i18n_bundle.Handler
	AuthProfile      *auth_profile.Handler
	Navigation       *navigation.Handler
	Style            *style.Handler
	Revision         *revision.Handler
	Commit           *commit.Handler
	Domain           *domain.Handler
	Backup           *backup.Handler
	Release          *release.Handler
}

func SetupRoutes(app *fiber.App, cfg *config.Config, handlers Handlers, authMiddleware httpmiddleware.AuthMiddleware, meter metric.Meter, logger *zap.Logger) {
	httpmiddleware.Register(app, cfg, meter, logger)
	if !cfg.App.IsProduction() {
		openapi.RegisterRoutes(app)
	}
	health.RegisterRoutes(app, health.Config{Service: cfg.App.Name, Version: cfg.App.Version, Env: cfg.App.Env})
	configuratorauth.RegisterPublicRoutes(app, handlers.ConfiguratorAuth)
	app.Get("/auth/session", authMiddleware.AuthMiddleware(), handlers.CurrentUser.Resolve(), handlers.Session.Current)
	api := app.Group("/api", authMiddleware.AuthMiddleware(), handlers.CurrentUser.Resolve())
	httpsession.RegisterRoutes(api, handlers.Session)
	v1 := api.Group("/v1")
	workspace.RegisterRoutes(v1, handlers.Workspace)
	integration.RegisterRoutes(v1, handlers.Integration)
	scoped := v1.Group("", handlers.Workspace.RequireWorkspace())
	project.RegisterRoutes(scoped, handlers.Project)
	tenant.RegisterRoutes(scoped, handlers.Tenant)
	environment.RegisterRoutes(scoped, handlers.Environment)
	folder.RegisterRoutes(scoped, handlers.Folder)
	domain_type.RegisterRoutes(scoped, handlers.Type)
	query.RegisterRoutes(scoped, handlers.Query)
	data_view.RegisterRoutes(scoped, handlers.DataView)
	composition.RegisterRoutes(scoped, handlers.Composition)
	store.RegisterRoutes(scoped, handlers.Store)
	stream.RegisterRoutes(scoped, handlers.Stream)
	update.RegisterRoutes(scoped, handlers.Update)
	mock.RegisterRoutes(scoped, handlers.Mock)
	component.RegisterRoutes(scoped, handlers.Component)
	action.RegisterRoutes(scoped, handlers.Action)
	filter.RegisterRoutes(scoped, handlers.Filter)
	converter.RegisterRoutes(scoped, handlers.Converter)
	computation.RegisterRoutes(scoped, handlers.Computation)
	vocab.RegisterRoutes(scoped, handlers.Vocab)
	i18n_bundle.RegisterRoutes(scoped, handlers.I18nBundle)
	auth_profile.RegisterRoutes(scoped, handlers.AuthProfile)
	navigation.RegisterRoutes(scoped, handlers.Navigation)
	style.RegisterRoutes(scoped, handlers.Style)
	revision.RegisterRoutes(scoped, handlers.Revision)
	commit.RegisterRoutes(scoped, handlers.Commit)
	domain.RegisterRoutes(scoped, handlers.Domain)
	backup.RegisterRoutes(scoped, handlers.Backup)
	release.RegisterRoutes(scoped, handlers.Release)
	app.Use(func(c *fiber.Ctx) error { return respond.WriteErrorResponse(c, respond.ErrRouteNotFound) })
}
