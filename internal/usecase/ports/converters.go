package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type ConvertersFilter struct {
	ProjectID uuid.UUID
	FolderID  *uuid.UUID
}

// ConvertersRepository defines persistence operations required by converter use cases.
type ConvertersRepository interface {
	Create(ctx context.Context, converter *entities.RConverter) (*entities.RConverter, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RConverter, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RConverter, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RConverter, error)
	List(ctx context.Context, filter ConvertersFilter) ([]*entities.RConverter, error)
	Update(ctx context.Context, converter *entities.RConverter) (*entities.RConverter, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter ConvertersFilter) (int64, error)
}
