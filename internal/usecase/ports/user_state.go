package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type UserStateRepository interface {
	LockUserState(context.Context, entities.UserStateScope, string) error
	UserStateScopes(context.Context, entities.UserStateScope) ([]string, error)
	UserStateUsers(context.Context, entities.UserStateScope, string, string, int) ([]entities.UserStateUser, error)
	UserStateRevision(context.Context, entities.UserStateScope, string) (int64, error)
	ReadUserState(context.Context, entities.UserStateScope, string) ([]entities.UserStateEntry, error)
	WriteUserState(context.Context, entities.UserStateScope, string, string, entities.UserStateEntry) (entities.UserStateEntry, error)
	ResetUserState(context.Context, entities.UserStateScope, string) (int64, error)
	RecordUserStateAdminAction(context.Context, entities.UserStateScope, string, string, string, []string, int64) error
}
