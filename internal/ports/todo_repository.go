package ports

import (
	"context"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
)

type TodoRepository interface {
	CreateTodo(ctx context.Context, todo *entities.Todo) (*entities.Todo, error)
}
