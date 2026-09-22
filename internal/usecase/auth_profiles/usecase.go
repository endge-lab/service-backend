package auth_profiles

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

var Definition = documents.Definition{Collection: "auth-profiles"}

// UseCase координирует сценарии работы с профилями аутентификации.
type UseCase struct {
	lifecycle  *documents.Lifecycle
	repository ports.AuthProfileRepository
}

// NewUseCase создаёт use case для работы с профилями аутентификации.
func NewUseCase(lifecycle *documents.Lifecycle, repository ports.AuthProfileRepository) *UseCase {
	return &UseCase{lifecycle: lifecycle, repository: repository}
}

// List возвращает документы ресурса «Профили аутентификации» с учётом фильтров.
func (s *UseCase) List(ctx context.Context, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.lifecycle.List(ctx, Definition, s.repository, filter)
}

// Get возвращает документ ресурса «Профили аутентификации» по identity.
func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.lifecycle.Get(ctx, Definition, s.repository, identity, includeDeleted)
}

// Create создаёт документ ресурса «Профили аутентификации» и фиксирует новую ревизию.
func (s *UseCase) Create(ctx context.Context, input documents.CreateInput) (*entities.Document, error) {
	if _, _, err := shared.AdminContext(ctx); err != nil {
		return nil, err
	}
	return s.lifecycle.Create(ctx, Definition, s.repository, input)
}

// Patch частично обновляет документ ресурса «Профили аутентификации» с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input documents.PatchInput, expected int) (*entities.Document, error) {
	if _, _, err := shared.AdminContext(ctx); err != nil {
		return nil, err
	}
	return s.lifecycle.Patch(ctx, Definition, s.repository, identity, input, expected)
}

// Delete мягко удаляет документ ресурса «Профили аутентификации» с проверкой ожидаемой ревизии.
func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	if _, _, err := shared.AdminContext(ctx); err != nil {
		return nil, err
	}
	return s.lifecycle.Delete(ctx, Definition, s.repository, identity, expected)
}

// Restore восстанавливает мягко удалённый документ ресурса «Профили аутентификации».
func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	if _, _, err := shared.AdminContext(ctx); err != nil {
		return nil, err
	}
	return s.lifecycle.Restore(ctx, Definition, s.repository, identity, expected)
}

var _ documents.ResourceUseCase = (*UseCase)(nil)
