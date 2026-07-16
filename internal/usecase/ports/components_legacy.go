package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type ComponentsLegacyFilter struct {
	ProjectID     uuid.UUID
	FolderID      *uuid.UUID
	ComponentType *entities.RComponentLegacyType
}

// ComponentsLegacyRepository defines persistence operations required by component use cases.
type ComponentsLegacyRepository interface {
	Create(ctx context.Context, component *entities.RComponentLegacy) (*entities.RComponentLegacy, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RComponentLegacy, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RComponentLegacy, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RComponentLegacy, error)
	List(ctx context.Context, filter ComponentsLegacyFilter) ([]*entities.RComponentLegacy, error)
	Update(ctx context.Context, component *entities.RComponentLegacy) (*entities.RComponentLegacy, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter ComponentsLegacyFilter) (int64, error)
}
