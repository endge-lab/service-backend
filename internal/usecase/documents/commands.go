package documents

import (
	"context"
	"encoding/json"
	"time"

	configurationdomain "github.com/endge-lab/service-backend/internal/domain/configuration"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// Create создаёт документ и фиксирует его начальную ревизию.
func (s *Lifecycle) Create(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, input CreateInput) (result *entities.Document, err error) {
	values, err := input.values()
	if err != nil {
		return nil, err
	}
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if err = validateCollection(definition.Collection); err != nil {
		return nil, err
	}
	if err = rejectReadOnly(values); err != nil {
		return nil, err
	}
	configurationdomain.RemoveLegacySSEFromDocument(definition.Collection, values)
	normalizeFolderInput(definition.Collection, values)
	if err = validateDocument(definition.Collection, values); err != nil {
		return nil, err
	}
	document := documentFromInput(definition.Collection, scope.Workspace.ID, values, current.User.ID)
	folderID, err := s.resolveFolder(ctx, scope, definition.Collection, values)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		created, txErr := repository.Insert(txctx, document, folderID)
		if txErr != nil {
			return txErr
		}
		if txErr = s.replaceStructuredRelations(txctx, *created); txErr != nil {
			return txErr
		}
		if _, txErr = s.history.RecordDocument(txctx, *created, "create", nil); txErr != nil {
			return txErr
		}
		result, txErr = repository.Get(txctx, scope.Workspace.ID, created.Identity, true)
		return txErr
	})
	return result, shared.MapConflict(err)
}

// Patch частично обновляет документ с проверкой ожидаемой ревизии.
func (s *Lifecycle) Patch(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, identity string, input PatchInput, expected int) (result *entities.Document, err error) {
	patch, err := input.values()
	if err != nil {
		return nil, err
	}
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	if err = rejectReadOnly(patch); err != nil {
		return nil, err
	}
	configurationdomain.RemoveLegacySSEFromDocument(definition.Collection, patch)
	normalizeFolderInput(definition.Collection, patch)
	existing, err := repository.Get(ctx, scope.Workspace.ID, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if definition.Collection == "folders" && isSystemFolder(*existing) {
		return nil, domainerrors.Conflict("system_folder_immutable", "Root and system folders cannot be changed")
	}
	next := applyPatch(*existing, patch, current.User.ID)
	if err = validateDocument(definition.Collection, documentAsInput(next)); err != nil {
		return nil, err
	}
	if checksumContent(*existing) == checksumContent(next) {
		return existing, nil
	}
	folderID, err := s.resolveDocumentFolder(ctx, scope, next)
	if err != nil {
		return nil, err
	}
	if definition.Collection == "folders" && folderID != nil {
		if *folderID == existing.ID {
			return nil, domainerrors.InvalidInput("folder_self_parent", "Folder cannot be its own parent")
		}
		cycle, cycleErr := s.documents.FolderWouldCycle(ctx, scope.Workspace.ID, existing.ID, *folderID)
		if cycleErr != nil {
			return nil, cycleErr
		}
		if cycle {
			return nil, domainerrors.InvalidInput("folder_cycle", "Folder parent would create a cycle")
		}
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := repository.Update(txctx, next, expected, folderID)
		if txErr != nil {
			return txErr
		}
		if txErr = s.replaceStructuredRelations(txctx, *updated); txErr != nil {
			return txErr
		}
		_, txErr = s.history.RecordDocument(txctx, *updated, "update", nil)
		result = updated
		return txErr
	})
	return result, shared.MapConflict(err)
}

// Delete мягко удаляет документ с проверкой ожидаемой ревизии.
func (s *Lifecycle) Delete(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := repository.Get(ctx, scope.Workspace.ID, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt != nil {
		return existing, nil
	}
	if definition.Collection == "folders" && isSystemFolder(*existing) {
		return nil, domainerrors.Conflict("system_folder_immutable", "Root and system folders cannot be deleted")
	}
	now := time.Now().UTC()
	next := *existing
	next.DeletedAt, next.Active, next.UpdatedBy = &now, false, entities.Actor{ID: current.User.ID}
	folderID, err := s.resolveDocumentFolder(ctx, scope, next)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if definition.Collection == "folders" {
			txctx, err = s.history.BeginBatch(txctx, &scope.Workspace.ID, "delete", current.User.ID)
			if err != nil {
				return err
			}
			moved, txErr := s.documents.MoveFolderContents(txctx, scope.Workspace.ID, existing.ID, folderID, current.User.ID)
			if txErr != nil {
				return txErr
			}
			for _, movedDocument := range moved {
				if _, txErr = s.history.RecordDocument(txctx, movedDocument, "update", nil); txErr != nil {
					return txErr
				}
			}
		}
		updated, txErr := repository.Update(txctx, next, expected, folderID)
		if txErr != nil {
			return txErr
		}
		_, txErr = s.history.RecordDocument(txctx, *updated, "delete", nil)
		result = updated
		return txErr
	})
	return result, shared.MapConflict(err)
}

// Restore восстанавливает мягко удалённый документ.
func (s *Lifecycle) Restore(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, identity string, expected int) (result *entities.Document, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := repository.Get(ctx, scope.Workspace.ID, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if existing.DeletedAt == nil {
		return existing, nil
	}
	next := *existing
	next.DeletedAt, next.Active, next.UpdatedBy = nil, true, entities.Actor{ID: current.User.ID}
	if definition.Collection == "folders" {
		var data map[string]any
		_ = json.Unmarshal(next.Data, &data)
		delete(data, "parentIdentity")
		next.Data = mustJSON(data)
		next.FolderIdentity = nil
	}
	folderID, err := s.resolveDocumentFolder(ctx, scope, next)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := repository.Update(txctx, next, expected, folderID)
		if txErr != nil {
			return txErr
		}
		if txErr = s.replaceStructuredRelations(txctx, *updated); txErr != nil {
			return txErr
		}
		_, txErr = s.history.RecordDocument(txctx, *updated, "restore", nil)
		result = updated
		return txErr
	})
	return result, shared.MapConflict(err)
}
