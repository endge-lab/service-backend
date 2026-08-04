package backups

import (
	"context"
	"slices"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// List возвращает metadata backups без тяжёлого JSON содержимого.
func (s *UseCase) List(ctx context.Context, kind string, limit, offset int) ([]entities.SnapshotBackup, error) {
	scope, err := adminScope(ctx)
	if err != nil {
		return nil, err
	}
	if kind != "" && !slices.Contains([]string{"manual", "pre_import"}, kind) {
		return nil, domainerrors.InvalidInput("backup_kind_invalid", "kind must be manual or pre_import")
	}
	return s.backups.ListSnapshotBackups(ctx, scope.Workspace.ID, kind, false, limit, offset)
}

// Archive возвращает backups с содержимым для потокового ZIP-представления transport-слоя.
func (s *UseCase) Archive(ctx context.Context, kind string) ([]entities.SnapshotBackup, error) {
	scope, err := adminScope(ctx)
	if err != nil {
		return nil, err
	}
	if kind != "" && !slices.Contains([]string{"manual", "pre_import"}, kind) {
		return nil, domainerrors.InvalidInput("backup_kind_invalid", "kind must be manual or pre_import")
	}
	return s.backups.ListSnapshotBackups(ctx, scope.Workspace.ID, kind, true, 0, 0)
}

// Get возвращает backup по UUID или read-only alias last.
func (s *UseCase) Get(ctx context.Context, id string, includeData bool) (*entities.SnapshotBackup, error) {
	scope, err := adminScope(ctx)
	if err != nil {
		return nil, err
	}
	if id != "last" {
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			return nil, domainerrors.InvalidInput("backup_id_invalid", "id must be UUID or last")
		}
	}
	value, err := s.backups.GetSnapshotBackup(ctx, scope.Workspace.ID, id, includeData)
	return value, shared.MapNotFound(err)
}

func adminScope(ctx context.Context) (entities.WorkspaceAccess, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return scope, err
	}
	if !shared.CanAdmin(scope.Role) {
		return scope, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return scope, nil
}
