package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// SyncUserInput содержит данные для синхронизации пользователя с провайдером идентификации.
type SyncUserInput struct {
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
}

// UpsertCurrentUserInput содержит данные для создания или обновления текущего пользователя.
type UpsertCurrentUserInput struct {
	ProviderID  string
	Subject     string
	Issuer      string
	Username    string
	DisplayName string
}

// UserRepository задаёт синхронизацию идентификации, необходимую сценариям пользовательской сессии.
type UserRepository interface {
	SyncUserFromIdentity(ctx context.Context, input SyncUserInput) (*entities.User, error)
	UpsertCurrentUser(ctx context.Context, input UpsertCurrentUserInput) (*entities.User, error)
}
