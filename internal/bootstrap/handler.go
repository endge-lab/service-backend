package bootstrap

import (
	v1 "github.com/endge-lab/service-backend/internal/api/http/v1"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/project"
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
			v1.NewHandler,
		),
	)
}
