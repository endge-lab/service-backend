package entities

import (
	"context"
	"time"
)

const (
	// AccessTokenSecretPrefix делает персональный токен узнаваемым в заголовке и в сканерах секретов.
	AccessTokenSecretPrefix = "endge_pat_"
	AccessTokenScopeRead    = "read"
	AccessTokenScopeWrite   = "write"

	AccessTokenStatusActive  = "active"
	AccessTokenStatusExpired = "expired"
	AccessTokenStatusRevoked = "revoked"
)

const accessTokenContextKey contextKey = "access_token"

// AccessToken описывает персональный токен доступа без секрета.
// Секрет хранится только как SHA-256 и показывается владельцу один раз при выпуске.
type AccessToken struct {
	ID                   string
	Owner                Actor
	WorkspaceID          string
	WorkspaceIdentity    string
	WorkspaceDisplayName string
	Name                 string
	Description          string
	Scope                string
	CanCommit            bool
	CanDelete            bool
	TokenPrefix          string
	CreatedAt            time.Time
	ExpiresAt            time.Time
	LastUsedAt           *time.Time
	RevokedAt            *time.Time
	RevokedBy            *Actor
}

// Status вычисляет состояние токена на момент now; отзыв важнее истечения срока.
func (t AccessToken) Status(now time.Time) string {
	if t.RevokedAt != nil {
		return AccessTokenStatusRevoked
	}
	if !now.Before(t.ExpiresAt) {
		return AccessTokenStatusExpired
	}
	return AccessTokenStatusActive
}

// AccessTokenPrincipal — проверенный токен текущего запроса.
// Он сужает права владельца: один workspace, scope и отдельные флаги commit/delete.
type AccessTokenPrincipal struct {
	TokenID            string
	UserID             string
	WorkspaceID        string
	WorkspaceIdentity  string
	Scope              string
	CanCommit          bool
	CanDelete          bool
	ExpiresAt          time.Time
	OwnerPlatformAdmin bool
}

// CanWrite сообщает, разрешает ли scope токена изменять документы.
func (p AccessTokenPrincipal) CanWrite() bool { return p.Scope == AccessTokenScopeWrite }

// CapWorkspaceRole возвращает пересечение роли владельца в workspace и scope токена.
// Административная роль через токен никогда не выдаётся.
func (p AccessTokenPrincipal) CapWorkspaceRole(role string) string {
	switch role {
	case "":
		return ""
	case "viewer":
		return "viewer"
	}
	if p.CanWrite() {
		return "editor"
	}
	return "viewer"
}

func WithAccessToken(ctx context.Context, principal AccessTokenPrincipal) context.Context {
	return context.WithValue(ctx, accessTokenContextKey, principal)
}

// AccessTokenFromContext возвращает токен запроса; ok=false для cookie и JWT сессий.
func AccessTokenFromContext(ctx context.Context) (AccessTokenPrincipal, bool) {
	principal, ok := ctx.Value(accessTokenContextKey).(AccessTokenPrincipal)
	return principal, ok
}
