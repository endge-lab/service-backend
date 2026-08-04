package middleware

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type contextKey string

const WorkspaceHeader = "X-Endge-Workspace"

const (
	userIDKey      contextKey = "user_id"
	sessionIDKey   contextKey = "session_id"
	identityKey    contextKey = "identity"
	currentUserKey contextKey = "current_user"
)

type RequestIdentity struct {
	ProviderID    string
	Subject       string
	Issuer        string
	AuthUserID    string
	Username      string
	DisplayName   string
	Role          string
	SessionID     string
	App           string
	Platform      string
	Scope         []string
	Groups        []string
	PlatformAdmin bool
	ExpiresAt     string
}

func IdentityFromContext(ctx context.Context) (RequestIdentity, bool) {
	identity, ok := ctx.Value(identityKey).(RequestIdentity)
	return identity, ok
}

func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func GetSessionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey).(string)
	return id, ok
}

func CurrentUserFromContext(ctx context.Context) (*entities.User, bool) {
	user, ok := ctx.Value(currentUserKey).(*entities.User)
	return user, ok
}
