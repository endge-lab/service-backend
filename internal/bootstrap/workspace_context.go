package bootstrap

import (
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	workspace "github.com/endge-lab/service-backend/internal/api/http/v1/workspace"

	"go.uber.org/fx"
)

// WorkspaceContextModules wires the workspace resolver into HTTP scope middleware.
func WorkspaceContextModules() fx.Option {
	return fx.Provide(newWorkspaceContextMiddleware)
}

func newWorkspaceContextMiddleware(resolver workspace.UseCase) *httpmiddleware.WorkspaceContextMiddleware {
	return httpmiddleware.NewWorkspaceContextMiddleware(resolver)
}
