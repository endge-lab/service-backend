package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func ServiceUser(value sqlc.ServiceUser) *entities.User {
	return &entities.User{ID: value.ID.String(), AuthUserID: value.AuthUserID, Username: value.Username, DisplayName: value.DisplayName, Role: value.Role, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
