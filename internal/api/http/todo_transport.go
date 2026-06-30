package http

import (
	"time"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
)

// CreateTodoRequest contains the external payload for creating a Todo task.
type CreateTodoRequest struct {
	Title string `json:"title" validate:"required,min=1,max=160"`
}

// TodoResponse describes a Todo entity in the external HTTP contract.
type TodoResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	IsCompleted bool      `json:"isCompleted"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newTodoResponse(todo *entities.Todo) *TodoResponse {
	if todo == nil {
		return nil
	}

	return &TodoResponse{
		ID:          todo.ID,
		Title:       todo.Title,
		IsCompleted: todo.IsCompleted,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	}
}
