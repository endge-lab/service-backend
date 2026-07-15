package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type ComponentsFilter struct {
	ProjectID     uuid.UUID
	FolderID      *uuid.UUID
	ComponentType *entities.ComponentType
}

type ComponentsRepository interface {
	Create(ctx context.Context, component *entities.Component) (*entities.Component, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Component, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Component, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Component, error)
	List(ctx context.Context, filter ComponentsFilter) ([]*entities.Component, error)
	Update(ctx context.Context, component *entities.Component) (*entities.Component, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter ComponentsFilter) (int64, error)
}
