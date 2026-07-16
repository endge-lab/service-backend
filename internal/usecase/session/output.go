package session

import "github.com/endge-lab/service-backend/internal/domain/entities"

type LoadSessionOutput struct {
	Session *entities.SessionInfo
	User    *entities.User
}
