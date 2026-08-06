package workspace_state

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// PlanReleaseRestore строит предварительный план операции без изменения состояния.
func (s *Coordinator) PlanReleaseRestore(ctx context.Context, identity string) (*entities.ImportPlan, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	if !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	bundle, err := s.releaseBundle(ctx, "restore_plan", scope.Workspace.ID, identity)
	if err != nil {
		return nil, err
	}
	return s.planExactRestore(ctx, scope, *bundle)
}

// RestoreRelease восстанавливает состояние рабочего пространства по релизу.
func (s *Coordinator) RestoreRelease(ctx context.Context, identity string, expected int64) (*entities.Commit, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	bundle, err := s.releaseBundle(ctx, "restore", scope.Workspace.ID, identity)
	if err != nil {
		return nil, err
	}
	return s.restoreBundle(ctx, *bundle, expected, "release_restore", "Restore release "+identity)
}

func (s *Coordinator) releaseBundle(ctx context.Context, operation, workspaceID, identity string) (*entities.PortableBundle, error) {
	release, err := s.repository.GetReleaseMetadata(ctx, workspaceID, identity)
	if err != nil {
		return nil, mapNotFound(err)
	}
	artifact, err := s.artifacts.Read(ctx, operation, workspaceID, *release)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var bundle entities.PortableBundle
	if err = json.Unmarshal(artifact.Data, &bundle); err != nil {
		return nil, domainerrors.Internal("release_snapshot_invalid", "Release snapshot is invalid")
	}
	return &bundle, nil
}
