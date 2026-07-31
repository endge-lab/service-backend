package bootstrap

import (
	httpapi "github.com/endge-lab/service-backend/internal/api/http"
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	"github.com/endge-lab/service-backend/internal/api/http/session"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component_legacy"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/domain"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	workspace "github.com/endge-lab/service-backend/internal/api/http/v1/workspace"
	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/internal/usecase/dependencies"

	"go.uber.org/fx"
)

func HandlerModules() fx.Option {
	return fx.Options(
		fx.Invoke(httpmiddleware.ConfigureTraceMiddleware),
		fx.Provide(
			httpobservability.NewHandlerMetrics,
			auth.NewResolver,
			func(service *dependencies.Dependencies) domain.UseCase { return service },
			fx.Annotate(httpmiddleware.NewAuthMiddleware, fx.As(new(httpmiddleware.AuthMiddleware))),
			fx.Annotate(session.NewHandler, fx.As(new(session.SHandler))),
			fx.Annotate(project.NewHandler, fx.As(new(project.PHandler))),
			fx.Annotate(folder.NewHandler, fx.As(new(folder.FHandler))),
			fx.Annotate(component_legacy.NewHandler, fx.As(new(component_legacy.CHandler))),
			fx.Annotate(converter.NewHandler, fx.As(new(converter.ConvHandler))),
			fx.Annotate(query.NewHandler, fx.As(new(query.QHandler))),
			fx.Annotate(data_view.NewHandler, fx.As(new(data_view.DVHandler))),
			fx.Annotate(domain.NewHandler, fx.As(new(domain.DHandler))),
			fx.Annotate(workspace.NewHandler, fx.As(new(workspace.WHandler))),
			httpapi.NewHandler,
		),
	)
}
