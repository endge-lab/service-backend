package workspace_state

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// PlanCommitRestore строит предварительный план операции без изменения состояния.
func (s *Coordinator) PlanCommitRestore(ctx context.Context, id string) (*entities.ImportPlan, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	if !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	commit, err := s.repository.GetCommit(ctx, scope.Workspace.ID, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	raw, err := s.repository.ExportWorkspace(ctx, scope.Workspace.ID, &commit.HeadSequence)
	if err != nil {
		return nil, err
	}
	var bundle entities.PortableBundle
	if err = json.Unmarshal(raw, &bundle); err != nil {
		return nil, err
	}
	return s.planExactRestore(ctx, scope, bundle)
}

// RestoreCommit восстанавливает состояние рабочего пространства по коммиту.
func (s *Coordinator) RestoreCommit(ctx context.Context, id string, expected int64) (*entities.Commit, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	commit, err := s.repository.GetCommit(ctx, scope.Workspace.ID, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	raw, err := s.repository.ExportWorkspace(ctx, scope.Workspace.ID, &commit.HeadSequence)
	if err != nil {
		return nil, err
	}
	var bundle entities.PortableBundle
	if err = json.Unmarshal(raw, &bundle); err != nil {
		return nil, err
	}
	return s.restoreBundle(ctx, bundle, expected, "commit_restore", "Restore commit "+id)
}

// Export экспортирует состояние рабочего пространства в переносимый пакет.
func (s *Coordinator) Export(ctx context.Context) (json.RawMessage, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	err = s.tx.WithinReadTransaction(ctx, func(txctx context.Context) error {
		latest, txErr := s.repository.LatestCommit(txctx, scope.Workspace.ID)
		if txErr != nil {
			return txErr
		}
		if latest.HeadSequence != scope.Workspace.HeadSequence {
			return domainerrors.Conflict("export_requires_clean_commit", "Workspace has uncommitted revisions")
		}
		result, txErr = s.repository.ExportWorkspace(txctx, scope.Workspace.ID, &latest.HeadSequence)
		return txErr
	})
	return result, err
}

// ExportLive возвращает текущее рабочее состояние с локальными server state-полями.
func (s *Coordinator) ExportLive(ctx context.Context) (json.RawMessage, error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	err = s.tx.WithinReadTransaction(ctx, func(txctx context.Context) error {
		var txErr error
		result, txErr = s.repository.ExportLiveWorkspace(txctx, scope.Workspace.ID)
		return txErr
	})
	return result, err
}
