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
	role, err := s.workspaces.WorkspaceRole(ctx, workspace.ID, current.User.ID, current.PlatformAdmin)
	if err != nil {
		return entities.WorkspaceAccess{}, err
	}
	if role == "" {
		return entities.WorkspaceAccess{}, domainerrors.Forbidden("workspace_forbidden", "Workspace access is forbidden")
	}
	return entities.WorkspaceAccess{Workspace: *workspace, Role: role}, nil
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

// Get возвращает рабочее пространство по identity.
func (s *UseCase) Get(ctx context.Context, identity string) (*entities.Workspace, error) {
	if _, err := s.Authorize(ctx, identity); err != nil {
		return nil, err
	}
	return s.workspaces.GetWorkspace(ctx, identity)
}
