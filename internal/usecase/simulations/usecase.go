package simulations

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

var Definition = documents.Definition{Collection: "simulations"}

// UseCase координирует сценарии работы с симуляциями.
type UseCase struct {
	lifecycle  *documents.Lifecycle
	repository ports.SimulationRepository
}

// NewUseCase создаёт use case для работы с симуляциями.
func NewUseCase(lifecycle *documents.Lifecycle, repository ports.SimulationRepository) *UseCase {
	return &UseCase{lifecycle: lifecycle, repository: repository}
}

// List возвращает документы ресурса «Симуляции» с учётом фильтров.
func (s *UseCase) List(ctx context.Context, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.lifecycle.List(ctx, Definition, s.repository, filter)
}

// Get возвращает документ ресурса «Симуляции» по identity.
func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.lifecycle.Get(ctx, Definition, s.repository, identity, includeDeleted)
}

// Create создаёт документ ресурса «Симуляции» и фиксирует новую ревизию.
func (s *UseCase) Create(ctx context.Context, input documents.CreateInput) (*entities.Document, error) {
	return s.lifecycle.Create(ctx, Definition, s.repository, input)
}

// Patch частично обновляет документ ресурса «Симуляции» с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input documents.PatchInput, expected int) (*entities.Document, error) {
	return s.lifecycle.Patch(ctx, Definition, s.repository, identity, input, expected)
}

// Delete мягко удаляет документ ресурса «Симуляции» с проверкой ожидаемой ревизии.
func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Delete(ctx, Definition, s.repository, identity, expected)
}

// Restore восстанавливает мягко удалённый документ ресурса «Симуляции».
func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Restore(ctx, Definition, s.repository, identity, expected)
}

var _ documents.ResourceUseCase = (*UseCase)(nil)
