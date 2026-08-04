package workspaces

import (
	"context"
	"slices"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// ListMemberships возвращает участников рабочего пространства и их роли.
func (s *UseCase) ListMemberships(ctx context.Context, identity string) ([]entities.Membership, error) {
	scope, err := s.Authorize(ctx, identity)
	if err != nil {
		return nil, err
	}
	if !shared.CanAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return s.workspaces.ListMemberships(ctx, scope.Workspace.ID)
}

// PutMembership создаёт или обновляет роль участника рабочего пространства.
func (s *UseCase) PutMembership(ctx context.Context, identity, userID, role string) (*entities.Membership, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := s.Authorize(ctx, identity)
	if err != nil {
		return nil, err
	}
	if !shared.CanAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	if !slices.Contains([]string{"viewer", "editor", "admin"}, role) {
		return nil, domainerrors.InvalidInput("membership_role_invalid", "role must be viewer, editor or admin")
	}
	return s.workspaces.PutMembership(ctx, scope.Workspace.ID, userID, role, current.User.ID)
}

// DeleteMembership удаляет участника из рабочего пространства.
func (s *UseCase) DeleteMembership(ctx context.Context, identity, userID string) error {
	scope, err := s.Authorize(ctx, identity)
	if err != nil {
		return err
	}
	if !shared.CanAdmin(scope.Role) {
		return domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return s.workspaces.DeleteMembership(ctx, scope.Workspace.ID, userID)
}
