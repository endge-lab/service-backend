package revisions

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// MaxListLimit — наибольший размер страницы истории ревизий.
const MaxListLimit = 500

// List возвращает страницу истории ревизий выбранного документа.
// Limit 0 сохраняет прежний контракт — вся история целиком.
func (s *UseCase) List(ctx context.Context, documentType, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	if err := revisionCollection(documentType); err != nil {
		return ports.RevisionPage{}, err
	}
	if options.Limit < 0 || options.Limit > MaxListLimit {
		return ports.RevisionPage{}, domainerrors.InvalidInput("limit_invalid", "limit must be between 1 and 500")
	}
	if options.Offset < 0 {
		return ports.RevisionPage{}, domainerrors.InvalidInput("offset_invalid", "offset must be non-negative")
	}
	scope, err := shared.Access(ctx)
	if err != nil {
		return ports.RevisionPage{}, err
	}
	return s.repository.ListRevisions(ctx, scope.Workspace.ID, documentType, identity, options)
}

// Get возвращает ревизию документа по идентификатору.
func (s *UseCase) Get(ctx context.Context, documentType, identity, revisionID string) (*entities.Revision, error) {
	if err := revisionCollection(documentType); err != nil {
		return nil, err
	}
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetRevision(ctx, scope.Workspace.ID, documentType, identity, revisionID)
	return value, shared.MapNotFound(err)
}

// revisionCollection определяет коллекцию документа из ревизии.
func revisionCollection(value string) error {
	for _, collection := range documents.Collections {
		if value == collection {
			return nil
		}
	}
	return domainerrors.WithDetails(domainerrors.InvalidInput("collection_unsupported", "Collection is not supported by this MVP"), map[string]any{"collection": value})
}
