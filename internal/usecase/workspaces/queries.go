package workspaces

import (
	"context"
	"errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// Authorize проверяет доступ пользователя к рабочему пространству.
func (s *UseCase) Authorize(ctx context.Context, identity string) (entities.WorkspaceAccess, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return entities.WorkspaceAccess{}, err
	}
	workspace, err := s.workspaces.GetWorkspace(ctx, strings.TrimSpace(identity))
	if errors.Is(err, ports.ErrNotFound) {
		return entities.WorkspaceAccess{}, domainerrors.NotFound("workspace_not_found", "Workspace not found")
	}
	if err != nil {
		return entities.WorkspaceAccess{}, err
	}
	platformAdmin := current.PlatformAdmin
	token, viaToken := entities.AccessTokenFromContext(ctx)
	if viaToken {
		// Токен привязан к одному workspace; роль владельца сужается scope токена.
		if workspace.ID != token.WorkspaceID {
			return entities.WorkspaceAccess{}, domainerrors.Forbidden("access_token_workspace_forbidden", "Access token is not valid for this workspace")
		}
		platformAdmin = token.OwnerPlatformAdmin
	}
	role, err := s.workspaces.WorkspaceRole(ctx, workspace.ID, current.User.ID, platformAdmin)
	if err != nil {
		return entities.WorkspaceAccess{}, err
	}
	ownerRole := ""
	if viaToken {
		ownerRole = role
		role = token.CapWorkspaceRole(role)
	}
	if role == "" {
		return entities.WorkspaceAccess{}, domainerrors.Forbidden("workspace_forbidden", "Workspace access is forbidden")
	}
	return entities.WorkspaceAccess{Workspace: *workspace, Role: role, OwnerRole: ownerRole}, nil
}

// List возвращает рабочие пространства, доступные текущему пользователю.
func (s *UseCase) List(ctx context.Context) ([]entities.Workspace, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	return s.workspaces.ListWorkspaces(ctx, current.User.ID, current.PlatformAdmin)
}

// ListAccess возвращает доступные workspace вместе с эффективной ролью пользователя.
func (s *UseCase) ListAccess(ctx context.Context) ([]entities.WorkspaceAccess, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	if _, viaToken := entities.AccessTokenFromContext(ctx); viaToken {
		return s.listTokenAccess(ctx)
	}
	items, err := s.workspaces.ListWorkspaces(ctx, current.User.ID, current.PlatformAdmin)
	if err != nil {
		return nil, err
	}
	result := make([]entities.WorkspaceAccess, 0, len(items))
	for _, workspace := range items {
		role, roleErr := s.workspaces.WorkspaceRole(ctx, workspace.ID, current.User.ID, current.PlatformAdmin)
		if roleErr != nil {
			return nil, roleErr
		}
		if role == "platform_admin" {
			role = "admin"
		}
		result = append(result, entities.WorkspaceAccess{Workspace: workspace, Role: role})
	}
	return result, nil
}

// listTokenAccess показывает агенту только workspace токена с уже суженной ролью.
func (s *UseCase) listTokenAccess(ctx context.Context) ([]entities.WorkspaceAccess, error) {
	token, _ := entities.AccessTokenFromContext(ctx)
	access, err := s.Authorize(ctx, token.WorkspaceIdentity)
	if code := domainerrors.CodeOf(err); code == "workspace_forbidden" || code == "workspace_not_found" {
		return []entities.WorkspaceAccess{}, nil
	}
	if err != nil {
		return nil, err
	}
	return []entities.WorkspaceAccess{access}, nil
}

// ListArchive returns Workspace tombstones visible to the current actor.
func (s *UseCase) ListArchive(ctx context.Context) ([]entities.WorkspaceAccess, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.workspaces.ListDeletedWorkspaces(ctx, current.User.ID, current.PlatformAdmin)
	if err != nil {
		return nil, err
	}
	result := make([]entities.WorkspaceAccess, 0, len(items))
	for _, workspace := range items {
		role, roleErr := s.workspaces.DeletedWorkspaceRole(ctx, workspace.ID, current.User.ID, current.PlatformAdmin)
		if roleErr != nil {
			return nil, roleErr
		}
		if role == "platform_admin" {
			role = "admin"
		}
		result = append(result, entities.WorkspaceAccess{Workspace: workspace, Role: role})
	}
	return result, nil
}

// Get возвращает рабочее пространство по identity.
func (s *UseCase) Get(ctx context.Context, identity string) (*entities.Workspace, error) {
	if _, err := s.Authorize(ctx, identity); err != nil {
		return nil, err
	}
	return s.workspaces.GetWorkspace(ctx, identity)
}
