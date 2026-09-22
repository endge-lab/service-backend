package folders

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

var Definition = documents.Definition{Collection: "folders"}

// UseCase координирует сценарии работы с папками.
type UseCase struct {
	lifecycle  *documents.Lifecycle
	repository ports.FolderRepository
}

// NewUseCase создаёт use case для работы с папками.
func NewUseCase(lifecycle *documents.Lifecycle, repository ports.FolderRepository) *UseCase {
	return &UseCase{lifecycle: lifecycle, repository: repository}
}

// List возвращает документы ресурса «Папки» с учётом фильтров.
func (s *UseCase) List(ctx context.Context, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.lifecycle.List(ctx, Definition, s.repository, filter)
}

// Get возвращает документ ресурса «Папки» по identity.
func (s *UseCase) Get(ctx context.Context, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.lifecycle.Get(ctx, Definition, s.repository, identity, includeDeleted)
}

// Create создаёт документ ресурса «Папки» и фиксирует новую ревизию.
func (s *UseCase) Create(ctx context.Context, input documents.CreateInput) (*entities.Document, error) {
	return s.lifecycle.Create(ctx, Definition, s.repository, input)
}

// Patch частично обновляет документ ресурса «Папки» с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input documents.PatchInput, expected int) (*entities.Document, error) {
	return s.lifecycle.Patch(ctx, Definition, s.repository, identity, input, expected)
}

// Delete мягко удаляет документ ресурса «Папки» с проверкой ожидаемой ревизии.
func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Delete(ctx, Definition, s.repository, identity, expected)
}

// Restore восстанавливает мягко удалённый документ ресурса «Папки».
func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return s.lifecycle.Restore(ctx, Definition, s.repository, identity, expected)
}

var _ documents.ResourceUseCase = (*UseCase)(nil)
