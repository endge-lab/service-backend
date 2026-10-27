package ports

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// CreateAccessTokenInput содержит только хэш секрета; сам секрет в persistence не передаётся.
type CreateAccessTokenInput struct {
	UserID      string
	WorkspaceID string
	Name        string
	Description string
	Scope       string
	CanCommit   bool
	CanDelete   bool
	TokenPrefix string
	SecretHash  []byte
	ExpiresAt   time.Time
}

// AccessTokenCredential — запись для проверки токена: хэш, ограничения и владелец.
type AccessTokenCredential struct {
	Token      entities.AccessToken
	SecretHash []byte
	Owner      entities.User
}

type AccessTokenRepository interface {
	CreateAccessToken(context.Context, CreateAccessTokenInput) (*entities.AccessToken, error)
	// ListAccessTokens возвращает токены владельца или все токены при userID == nil.
	ListAccessTokens(ctx context.Context, userID *string) ([]entities.AccessToken, error)
	GetAccessToken(context.Context, string) (*entities.AccessToken, error)
	FindAccessTokenCredential(ctx context.Context, secretHash []byte) (*AccessTokenCredential, error)
	// RevokeAccessToken идемпотентен: уже отозванный токен сохраняет первое время отзыва.
	RevokeAccessToken(ctx context.Context, id, actorID string) error
	// TouchAccessToken обновляет last_used_at не чаще, чем раз в minInterval.
	TouchAccessToken(ctx context.Context, id string, minInterval time.Duration) error
}
