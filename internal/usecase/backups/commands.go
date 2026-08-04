package backups

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// Create сохраняет бессрочный manual backup текущего live-состояния.
func (s *UseCase) Create(ctx context.Context, description *string) (result *entities.SnapshotBackup, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if !shared.CanAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	if description != nil {
		value := strings.TrimSpace(*description)
		if len(value) > 1000 {
			return nil, domainerrors.InvalidInput("backup_description_too_long", "description must not exceed 1000 characters")
		}
		if value == "" {
			description = nil
		} else {
			description = &value
		}
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if txErr := s.backups.LockWorkspaceSnapshot(txctx, scope.Workspace.ID); txErr != nil {
			return txErr
		}
		raw, txErr := s.portable.ExportWorkspace(txctx, scope.Workspace.ID, nil)
		if txErr != nil {
			return txErr
		}
		sum := sha256.Sum256(raw)
		result, txErr = s.backups.CreateSnapshotBackup(txctx, entities.SnapshotBackup{
			ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, Kind: "manual", Description: description,
			SchemaVersion: 1, Checksum: hex.EncodeToString(sum[:]), Data: raw, CreatedBy: entities.Actor{ID: current.User.ID},
		})
		return txErr
	})
	return result, err
}
