package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type AccessGrantInput struct {
	UserID      string
	ScopeType   string
	WorkspaceID *string
	Role        string
	ActorID     string
}

type AccessGrantListInput struct {
	ScopeType   string
	WorkspaceID *string
	UserID      *string
	Query       string
	Cursor      *entities.AccessGrantCursor
	Limit       int
}

type ServiceUserSearchInput struct {
	Query  string
	Cursor *entities.AccessGrantCursor
	Limit  int
}

type AccessControlRepository interface {
	GetExternalAccessState(context.Context, string) (*entities.ExternalAccessState, error)
	HasExternalAccessHistory(context.Context) (bool, error)
	ReplaceExternalAccess(context.Context, string, []AccessGrantInput, entities.ExternalAccessState) error
	LockBootstrap(context.Context) error
	HasPlatformAdmins(context.Context) (bool, error)
	IsPlatformAdmin(context.Context, string) (bool, error)
	CountHumanUsers(context.Context) (int, error)
	SearchServiceUsers(context.Context, ServiceUserSearchInput) (entities.ServiceUserPage, error)
	GetAccessGrant(context.Context, string) (*entities.AccessGrant, error)
	ListAccessGrants(context.Context, AccessGrantListInput) (entities.AccessGrantPage, error)
	UpsertAccessGrant(context.Context, AccessGrantInput) (*entities.AccessGrant, bool, error)
	DeleteAccessGrant(context.Context, string) error
	CountPlatformAdmins(context.Context) (int, error)
}
