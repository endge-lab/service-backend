package ports

import (
	"context"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
)

type SyncUserInput struct {
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
}

type UserRepository interface {
	SyncUserFromIdentity(ctx context.Context, input SyncUserInput) (*entities.User, error)
}
