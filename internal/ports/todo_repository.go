package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type TodoRepository interface {
	CreateTodo(ctx context.Context, todo *entities.Todo) (*entities.Todo, error)
}
