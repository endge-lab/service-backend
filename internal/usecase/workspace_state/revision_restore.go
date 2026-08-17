package workspace_state

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// RestoreRevision восстанавливает документ до выбранной ревизии.
func (s *Coordinator) RestoreRevision(ctx context.Context, kind, identity, id string, expected int) (result *entities.Document, err error) {
	if !slices.Contains(Collections, kind) {
		return nil, unsupported(kind)
	}
	current, scope, err := s.writeContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, preconditionError()
	}
	existing, err := s.repository.GetDocument(ctx, scope.Workspace.ID, kind, identity, true)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, revisionConflict()
	}
	if kind == "folders" {
		var folderData map[string]any
		_ = json.Unmarshal(existing.Data, &folderData)
		if boolValue(folderData["isRoot"]) || existing.ManagedBy == "system" {
			return nil, domainerrors.Conflict("system_folder_immutable", "Root and system folders cannot be restored from revisions")
		}
	}
	revision, err := s.repository.GetRevision(ctx, scope.Workspace.ID, kind, identity, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var target entities.Document
	if err = json.Unmarshal(revision.Snapshot, &target); err != nil {
		return nil, domainerrors.Internal("revision_snapshot_invalid", "Revision snapshot is invalid")
	}
	target.ID = existing.ID
	target.WorkspaceID = existing.WorkspaceID
	target.Revision = existing.Revision
	target.UpdatedBy = entities.Actor{ID: current.User.ID}
	folderID, err := s.resolveDocumentFolder(ctx, scope, target)
	if err != nil && kind != "folders" && errors.Is(err, ports.ErrNotFound) {
		rootIdentity := entities.RootFolderIdentity(target.Type)
		target.FolderIdentity = &rootIdentity
		folderID, err = s.resolveDocumentFolder(ctx, scope, target)
	}
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, e := s.repository.UpdateDocument(txctx, target, expected, folderID)
		if e != nil {
			return e
		}
		if e = s.replaceStructuredRelations(txctx, *updated); e != nil {
			return e
		}
		_, e = s.recordRevision(txctx, *updated, "restore", &revision.ID)
		result = updated
		return e
	})
	return result, mapConflict(err)
}
