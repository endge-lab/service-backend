package middleware

import "context"

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	sessionIDKey contextKey = "session_id"
	identityKey  contextKey = "identity"
)

type RequestIdentity struct {
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
