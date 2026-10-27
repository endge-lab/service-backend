package access_token

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// CreateRequest — выпуск токена для себя. expiresAt обязателен и не позже, чем через
// ACCESS_TOKEN_MAX_TTL (по умолчанию 30 дней). commit и delete требуют scope write.
type CreateRequest struct {
	WorkspaceIdentity string    `json:"workspaceIdentity" validate:"required,max=160" example:"afl"`
	Name              string    `json:"name" validate:"required,max=120" example:"Codex: инспектор рейса"`
	Description       string    `json:"description,omitempty" validate:"omitempty,max=1000" example:"Правка Query и DataView инспектора"`
	Scope             string    `json:"scope,omitempty" validate:"omitempty,oneof=read write" enums:"read,write" default:"read" example:"read"`
	Commit            bool      `json:"commit,omitempty" example:"false"`
	Delete            bool      `json:"delete,omitempty" example:"false"`
	ExpiresAt         time.Time `json:"expiresAt" validate:"required" example:"2026-11-02T10:00:00Z" format:"date-time"`
}

// TokenResponse описывает токен без секрета.
type TokenResponse struct {
	ID                   string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	Owner                entities.Actor  `json:"owner"`
	WorkspaceIdentity    string          `json:"workspaceIdentity" example:"afl"`
	WorkspaceDisplayName string          `json:"workspaceDisplayName" example:"АФЛ"`
	Name                 string          `json:"name" example:"Codex: инспектор рейса"`
	Description          string          `json:"description,omitempty"`
	Scope                string          `json:"scope" enums:"read,write" example:"read"`
	Commit               bool            `json:"commit" example:"false"`
	Delete               bool            `json:"delete" example:"false"`
	TokenPrefix          string          `json:"tokenPrefix" example:"endge_pat_Ab3dE6gH"`
	Status               string          `json:"status" enums:"active,expired,revoked" example:"active"`
	CreatedAt            time.Time       `json:"createdAt" format:"date-time"`
	ExpiresAt            time.Time       `json:"expiresAt" format:"date-time"`
	LastUsedAt           *time.Time      `json:"lastUsedAt,omitempty" format:"date-time"`
	RevokedAt            *time.Time      `json:"revokedAt,omitempty" format:"date-time"`
	RevokedBy            *entities.Actor `json:"revokedBy,omitempty"`
}

// CreateResponse возвращает секрет единственный раз; повторно получить его нельзя.
type CreateResponse struct {
	Token       string        `json:"token" example:"endge_pat_Ab3dE6gH..."`
	AccessToken TokenResponse `json:"accessToken"`
}

type ListResponse struct {
	Items         []TokenResponse `json:"items"`
	MaxTTLSeconds int64           `json:"maxTtlSeconds" example:"2592000"`
}

func newTokenResponse(value entities.AccessToken, now time.Time) TokenResponse {
	return TokenResponse{
		ID: value.ID, Owner: value.Owner, WorkspaceIdentity: value.WorkspaceIdentity, WorkspaceDisplayName: value.WorkspaceDisplayName,
		Name: value.Name, Description: value.Description, Scope: value.Scope, Commit: value.CanCommit, Delete: value.CanDelete,
		TokenPrefix: value.TokenPrefix, Status: value.Status(now), CreatedAt: value.CreatedAt, ExpiresAt: value.ExpiresAt,
		LastUsedAt: value.LastUsedAt, RevokedAt: value.RevokedAt, RevokedBy: value.RevokedBy,
	}
}
