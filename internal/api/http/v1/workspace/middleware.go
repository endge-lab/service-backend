package workspace

import (
	"strings"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
)

// RequireWorkspace проверяет workspace header, RBAC и добавляет access scope в context.
func (h *Handler) RequireWorkspace() fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity := strings.TrimSpace(c.Get(middleware.WorkspaceHeader))
		if identity == "" {
			return respond.WriteErrorResponse(c, domainerrors.InvalidInput("workspace_required", "X-Endge-Workspace header is required"))
		}
		access, err := h.usecase.Authorize(c.UserContext(), identity)
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		c.SetUserContext(entities.WithWorkspaceAccess(c.UserContext(), access))
		return c.Next()
	}
}
