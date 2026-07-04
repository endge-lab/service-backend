package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type TodoRepository interface {
	Create(ctx context.Context, todo *entities.Todo) (*entities.Todo, error)
	GetByID(ctx context.Context, id string) (*entities.Todo, error)
	Update(ctx context.Context, todo *entities.Todo) (*entities.Todo, error)
	Delete(ctx context.Context, id string) error
}
