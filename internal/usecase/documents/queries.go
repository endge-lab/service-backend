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
