package documents

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// ResourceUseCase задаёт общий CRUD-контракт resource application use case.
type ResourceUseCase interface {
	List(context.Context, ports.DocumentFilter) ([]entities.Document, error)
	Get(context.Context, string, bool) (*entities.Document, error)
	Create(context.Context, CreateInput) (*entities.Document, error)
	Patch(context.Context, string, PatchInput, int) (*entities.Document, error)
	Delete(context.Context, string, int) (*entities.Document, error)
	Restore(context.Context, string, int) (*entities.Document, error)
}
