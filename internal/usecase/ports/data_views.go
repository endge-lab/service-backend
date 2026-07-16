package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type DataViewsFilter struct {
	ProjectID uuid.UUID
	FolderID  *uuid.UUID
	QueryID   *uuid.UUID
}

// DataViewsRepository defines persistence operations required by data-view use cases.
type DataViewsRepository interface {
	Create(ctx context.Context, dataView *entities.RDataView) (*entities.RDataView, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RDataView, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RDataView, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RDataView, error)
	List(ctx context.Context, filter DataViewsFilter) ([]*entities.RDataView, error)
	Update(ctx context.Context, dataView *entities.RDataView) (*entities.RDataView, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter DataViewsFilter) (int64, error)
}
