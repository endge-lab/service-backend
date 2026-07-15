package bootstrap

import (
	v1 "github.com/endge-lab/service-backend/internal/api/http/v1"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	session "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/internal/middleware"

	"go.uber.org/fx"
)

func HandlerModules() fx.Option {
	return fx.Options(
		fx.Provide(
			auth.NewResolver,
			fx.Annotate(middleware.NewAuthMiddleware, fx.As(new(middleware.AuthMiddleware))),
			fx.Annotate(session.NewHandler, fx.As(new(session.SHandler))),
			fx.Annotate(project.NewHandler, fx.As(new(project.PHandler))),
			fx.Annotate(folder.NewHandler, fx.As(new(folder.FHandler))),
			fx.Annotate(component.NewHandler, fx.As(new(component.CHandler))),
			fx.Annotate(converter.NewHandler, fx.As(new(converter.ConvHandler))),
			fx.Annotate(query.NewHandler, fx.As(new(query.QHandler))),
			fx.Annotate(data_view.NewHandler, fx.As(new(data_view.DVHandler))),
			v1.NewHandler,
		),
	)
}
