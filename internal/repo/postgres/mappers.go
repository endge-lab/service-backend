package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func mapTodo(todo sqlc.Todo) *entities.Todo {
	return &entities.Todo{
		ID:          todo.ID.String(),
		Title:       todo.Title,
		IsCompleted: todo.IsCompleted,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	}
}

func mapServiceUser(user sqlc.ServiceUser) *entities.User {
	return &entities.User{
		ID:          user.ID.String(),
		AuthUserID:  user.AuthUserID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
