package components

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

var Definition = documents.Definition{Collection: "components"}

// UseCase координирует сценарии работы с компонентами.
type UseCase struct {
	lifecycle  *documents.Lifecycle
	repository ports.ComponentRepository
}

// NewUseCase создаёт use case для работы с компонентами.
func NewUseCase(lifecycle *documents.Lifecycle, repository ports.ComponentRepository) *UseCase {
	return &UseCase{lifecycle: lifecycle, repository: repository}
}

// List возвращает документы ресурса «Компоненты» с учётом фильтров.
func (s *UseCase) List(ctx context.Context, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.lifecycle.List(ctx, Definition, s.repository, filter)
}

// Get возвращает документ ресурса «Компоненты» по identity.
func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.lifecycle.Get(ctx, Definition, s.repository, identity, includeDeleted)
}

// Create создаёт документ ресурса «Компоненты» и фиксирует новую ревизию.
func (s *UseCase) Create(ctx context.Context, input documents.CreateInput) (*entities.Document, error) {
	return s.lifecycle.Create(ctx, Definition, s.repository, input)
}

// Patch частично обновляет документ ресурса «Компоненты» с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input documents.PatchInput, expected int) (*entities.Document, error) {
	return s.lifecycle.Patch(ctx, Definition, s.repository, identity, input, expected)
}

// Delete мягко удаляет документ ресурса «Компоненты» с проверкой ожидаемой ревизии.
func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Delete(ctx, Definition, s.repository, identity, expected)
}

// Restore восстанавливает мягко удалённый документ ресурса «Компоненты».
func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Restore(ctx, Definition, s.repository, identity, expected)
}

var _ documents.ResourceUseCase = (*UseCase)(nil)
