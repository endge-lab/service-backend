package documents

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

const maxMoveDocuments = 500

// MoveDocumentInput задаёт документ и ожидаемую ревизию для массового переноса.
type MoveDocumentInput struct {
	Collection       string
	Identity         string
	ExpectedRevision int
}

// MoveDocumentsInput задаёт атомарный перенос документов в одну папку.
type MoveDocumentsInput struct {
	Documents      []MoveDocumentInput
	FolderIdentity string
}

// MovedDocument содержит актуальное состояние документа после переноса.
type MovedDocument struct {
	Collection string
	Document   entities.Document
}

// MoveDocumentsResult содержит документы в исходном порядке и число фактических изменений.
type MoveDocumentsResult struct {
	Documents []MovedDocument
	Moved     int
}

type preparedDocumentMove struct {
	index    int
	document entities.Document
	folderID *string
}

// MoveDocuments атомарно переносит несколько документов в одну папку и пишет общий mutation batch.
func (s *Lifecycle) MoveDocuments(ctx context.Context, input MoveDocumentsInput) (result MoveDocumentsResult, err error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return result, err
	}

	folderIdentity := strings.TrimSpace(input.FolderIdentity)
	if folderIdentity == "" {
		return result, domainerrors.InvalidInput("folder_identity_required", "folderIdentity is required")
	}
	if len(folderIdentity) > 160 {
		return result, domainerrors.InvalidInput("folder_identity_too_long", "folderIdentity must not exceed 160 characters")
	}
	if len(input.Documents) == 0 || len(input.Documents) > maxMoveDocuments {
		return result, domainerrors.InvalidInput("move_documents_count_invalid", "documents must contain from 1 to 500 items")
	}

	seen := make(map[string]struct{}, len(input.Documents))
	for index := range input.Documents {
		item := &input.Documents[index]
		item.Collection = strings.TrimSpace(item.Collection)
		item.Identity = strings.TrimSpace(item.Identity)
		if err = validateCollection(item.Collection); err != nil {
			return result, err
		}
		if item.Collection == "folders" {
			return result, domainerrors.InvalidInput("folder_move_unsupported", "Folders must be moved through the folder lifecycle")
		}
		if err = validateIdentity(item.Identity); err != nil {
			return result, err
		}
		if item.ExpectedRevision <= 0 {
			return result, domainerrors.InvalidInput("expected_revision_invalid", "expectedRevision must be positive")
		}

		key := item.Collection + ":" + item.Identity
		if _, exists := seen[key]; exists {
			return result, domainerrors.WithDetails(
				domainerrors.InvalidInput("move_document_duplicate", "The same document cannot be moved twice"),
				map[string]any{"document": key},
			)
		}
		seen[key] = struct{}{}
	}

	result.Documents = make([]MovedDocument, len(input.Documents))
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		prepared := make([]preparedDocumentMove, 0, len(input.Documents))
		for index, item := range input.Documents {
			existing, getErr := s.documents.GetDocument(txctx, scope.Workspace.ID, item.Collection, item.Identity, true)
			if getErr != nil {
				return shared.MapNotFound(getErr)
			}
			if existing.Revision != item.ExpectedRevision {
				return domainerrors.WithDetails(shared.RevisionConflict(), map[string]any{
					"collection": item.Collection,
					"identity":   item.Identity,
				})
			}
			if existing.DeletedAt != nil {
				return domainerrors.WithDetails(
					domainerrors.InvalidInput("move_document_deleted", "Deleted documents cannot be moved"),
					map[string]any{"collection": item.Collection, "identity": item.Identity},
				)
			}

			next := *existing
			next.FolderIdentity = &folderIdentity
			next.UpdatedBy = entities.Actor{ID: current.User.ID}
			folderID, resolveErr := s.resolveDocumentFolder(txctx, scope, next)
			if resolveErr != nil {
				return resolveErr
			}

			result.Documents[index] = MovedDocument{Collection: item.Collection, Document: *existing}
			if checksumContent(*existing) == checksumContent(next) {
				continue
			}
			prepared = append(prepared, preparedDocumentMove{index: index, document: next, folderID: folderID})
		}

		if len(prepared) == 0 {
			return nil
		}

		txctx, err = s.history.BeginBatch(txctx, &scope.Workspace.ID, "move", current.User.ID)
		if err != nil {
			return err
		}
		for _, move := range prepared {
			updated, updateErr := s.documents.UpdateDocument(txctx, move.document, move.document.Revision, move.folderID)
			if updateErr != nil {
				return updateErr
			}
			if _, updateErr = s.history.RecordDocument(txctx, *updated, "update", nil); updateErr != nil {
				return updateErr
			}
			result.Documents[move.index].Document = *updated
			result.Moved++
		}
		return nil
	})
	if err != nil {
		return MoveDocumentsResult{}, shared.MapConflict(err)
	}
	return result, nil
}
