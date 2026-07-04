package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type LoadSessionService interface {
	Execute(ctx context.Context, input LoadSessionInput) (*LoadSessionOutput, error)
}

type LoadSessionInput struct {
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
	SessionID   string
	App         string
	Platform    string
	Scope       []string
	ExpiresAt   string
}

type LoadSessionOutput struct {
	Session *entities.SessionInfo
	User    *entities.User
}

type CreateTodoService interface {
	Execute(ctx context.Context, input CreateTodoInput) (*CreateTodoOutput, error)
}

type CreateTodoInput struct {
	Title string
}

type CreateTodoOutput struct {
	Todo *entities.Todo
}
