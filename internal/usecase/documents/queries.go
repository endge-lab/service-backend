package documents

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// List возвращает документы коллекции с учётом фильтров и прав доступа.
func (s *Lifecycle) List(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, filter ports.DocumentFilter) ([]entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	if err = validateCollection(definition.Collection); err != nil {
		return nil, err
	}
	if filter.Limit <= 0 || filter.Limit > 1000 {
		filter.Limit = 100
	}
	return repository.List(ctx, scope.Workspace.ID, filter)
}

// Get возвращает документ коллекции по identity.
func (s *Lifecycle) Get(ctx context.Context, definition Definition, repository ports.DocumentResourceRepository, identity string, includeDeleted bool) (*entities.Document, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	if err = validateCollection(definition.Collection); err != nil {
		return nil, err
	}
	value, err := repository.Get(ctx, scope.Workspace.ID, identity, includeDeleted)
	return value, shared.MapNotFound(err)
}

// ArchivePage is a cursor-ready page of tombstones from the current workspace.
type ArchivePage struct {
	Items      []entities.ArchivedDocument
	NextOffset *int
}

// ListArchive returns deleted generic documents without loading their payloads.
func (s *Lifecycle) ListArchive(ctx context.Context, limit, offset int) (ArchivePage, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return ArchivePage{}, err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.documents.ListArchivedDocuments(ctx, scope.Workspace.ID, limit+1, offset)
	if err != nil {
		return ArchivePage{}, err
	}
	page := ArchivePage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		next := offset + limit
		page.NextOffset = &next
	}
	return page, nil
}
