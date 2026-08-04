package integrations

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// List возвращает интеграции, доступные текущему пользователю.
func (s *UseCase) List(ctx context.Context, includeDeleted bool) ([]entities.Integration, error) {
	if _, err := shared.Actor(ctx); err != nil {
		return nil, err
	}
	return s.repository.ListIntegrations(ctx, includeDeleted)
}

// Get возвращает интеграцию по идентификатору.
func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Integration, error) {
	if _, err := shared.Actor(ctx); err != nil {
		return nil, err
	}
	value, err := s.repository.GetIntegration(ctx, identity, includeDeleted)
	return value, shared.MapNotFound(err)
}
