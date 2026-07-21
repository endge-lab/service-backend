package middleware

import (
	"context"
	"strings"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

const WorkspaceHeader = "X-Endge-Workspace"

type WorkspaceResolver interface {
	GetByIdentity(context.Context, string) (*entities.RWorkspace, error)
}

// WorkspaceContextMiddleware resolves and attaches workspace scopes to requests.
type WorkspaceContextMiddleware struct {
	resolver WorkspaceResolver
}

// NewWorkspaceContextMiddleware creates middleware with its resolver dependency.
func NewWorkspaceContextMiddleware(resolver WorkspaceResolver) *WorkspaceContextMiddleware {
	return &WorkspaceContextMiddleware{resolver: resolver}
}

// RequireWorkspace resolves X-Endge-Workspace and attaches its UUID to UserContext.
func (m *WorkspaceContextMiddleware) RequireWorkspace() fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity := strings.TrimSpace(c.Get(WorkspaceHeader))
		if identity == "" {
			return respond.WriteErrorResponse(c, apperrors.InvalidInput("workspace_required", "workspace header is required"))
		}

		if m == nil || m.resolver == nil {
			return respond.WriteErrorResponse(c, apperrors.Internal("internal_error", "workspace resolver is unavailable"))
		}

		workspace, err := m.resolver.GetByIdentity(c.UserContext(), identity)
		if err != nil {
			if domainerrors.CodeOf(err) == "not_found" {
				return respond.WriteErrorResponse(c, domainerrors.NotFound("workspace_not_found", "workspace not found"))
			}
			return respond.RespondDomainError(c, nil, err)
		}

		c.SetUserContext(entities.WithWorkspace(c.UserContext(), entities.WorkspaceScope{
			ID:       workspace.ID,
			Identity: workspace.Identity,
		}))
		return c.Next()
	}
}
