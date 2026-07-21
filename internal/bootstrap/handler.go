package bootstrap

import (
	httpapi "github.com/endge-lab/service-backend/internal/api/http"
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/session"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component_legacy"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	workspace "github.com/endge-lab/service-backend/internal/api/http/v1/workspace"
	"github.com/endge-lab/service-backend/internal/auth"

	"go.uber.org/fx"
)

func HandlerModules() fx.Option {
	return fx.Options(
		fx.Invoke(httpmiddleware.ConfigureTraceMiddleware),
		fx.Provide(
			auth.NewResolver,
			fx.Annotate(httpmiddleware.NewAuthMiddleware, fx.As(new(httpmiddleware.AuthMiddleware))),
			fx.Annotate(session.NewHandler, fx.As(new(session.SHandler))),
			fx.Annotate(project.NewHandler, fx.As(new(project.PHandler))),
			fx.Annotate(folder.NewHandler, fx.As(new(folder.FHandler))),
			fx.Annotate(component_legacy.NewHandler, fx.As(new(component_legacy.CHandler))),
			fx.Annotate(converter.NewHandler, fx.As(new(converter.ConvHandler))),
			fx.Annotate(query.NewHandler, fx.As(new(query.QHandler))),
			fx.Annotate(data_view.NewHandler, fx.As(new(data_view.DVHandler))),
			fx.Annotate(workspace.NewHandler, fx.As(new(workspace.WHandler))),
			httpapi.NewHandler,
		),
	)
}
