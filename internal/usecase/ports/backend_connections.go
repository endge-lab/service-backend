package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// BackendConnectionRepository задаёт глобальный, не привязанный к Workspace каталог backend.
type BackendConnectionRepository interface {
	ListBackendConnections(context.Context) ([]entities.BackendConnection, error)
	InsertBackendConnection(context.Context, entities.BackendConnection) (*entities.BackendConnection, error)
	DeleteBackendConnection(context.Context, string) error
}
