package revisions

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// List возвращает историю ревизий выбранного документа.
func (s *UseCase) List(ctx context.Context, documentType, identity string) ([]entities.Revision, error) {
	if err := revisionCollection(documentType); err != nil {
		return nil, err
	}
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	return s.repository.ListRevisions(ctx, scope.Workspace.ID, documentType, identity)
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
