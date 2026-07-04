package bootstrap

import (
	v1 "github.com/endge-lab/service-backend/internal/api/http/v1"
	session "github.com/endge-lab/service-backend/internal/api/http/v1/session"
	todo "github.com/endge-lab/service-backend/internal/api/http/v1/todo"
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
			fx.Annotate(todo.NewHandler, fx.As(new(todo.THandler))),
			v1.NewHandler,
		),
	)
}
