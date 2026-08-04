package entities

import "context"

type contextKey string

const currentActorContextKey contextKey = "current_actor"
const workspaceAccessContextKey contextKey = "workspace_access"

type CurrentActor struct {
	User          *User
	PlatformAdmin bool
}

type WorkspaceAccess struct {
	Workspace Workspace
	Role      string
}

func WithCurrentActor(ctx context.Context, actor CurrentActor) context.Context {
	return context.WithValue(ctx, currentActorContextKey, actor)
}
func CurrentActorFromContext(ctx context.Context) (CurrentActor, bool) {
	actor, ok := ctx.Value(currentActorContextKey).(CurrentActor)
	return actor, ok
}
func WithWorkspaceAccess(ctx context.Context, access WorkspaceAccess) context.Context {
	return context.WithValue(ctx, workspaceAccessContextKey, access)
}
func WorkspaceAccessFromContext(ctx context.Context) (WorkspaceAccess, bool) {
	access, ok := ctx.Value(workspaceAccessContextKey).(WorkspaceAccess)
	return access, ok
}
