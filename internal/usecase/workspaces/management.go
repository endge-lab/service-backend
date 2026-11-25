package workspaces

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// workspaceManagementActor проверяет флаг токена и возвращает права владельца
// для операций жизненного цикла workspace. Actor в context остаётся суженным.
func workspaceManagementActor(ctx context.Context) (entities.CurrentActor, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return current, err
	}
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		if !token.CanManageWorkspaces() {
			return entities.CurrentActor{}, domainerrors.Forbidden("access_token_workspace_management_forbidden", "Access token does not allow workspace management")
		}
		current.PlatformAdmin = token.OwnerPlatformAdmin
	}
	return current, nil
}

// authorizeWorkspaceManagement вызывается после workspaceManagementActor.
func (s *UseCase) authorizeWorkspaceManagement(ctx context.Context, identity string) (entities.WorkspaceAccess, error) {
	scope, err := s.Authorize(ctx, identity)
	if err != nil {
		return scope, err
	}
	role := scope.Role
	if _, viaToken := entities.AccessTokenFromContext(ctx); viaToken {
		role = scope.OwnerRole
	}
	if !shared.CanAdmin(role) {
		return entities.WorkspaceAccess{}, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return scope, nil
}
