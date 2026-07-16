package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type QueriesFilter struct {
	ProjectID uuid.UUID
	FolderID  *uuid.UUID
	QueryType *string
}

// QueriesRepository defines persistence operations required by query use cases.
type QueriesRepository interface {
	Create(ctx context.Context, query *entities.RQuery) (*entities.RQuery, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RQuery, error)
	GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entities.RQuery, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RQuery, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RQuery, error)
	List(ctx context.Context, filter QueriesFilter) ([]*entities.RQuery, error)
	Update(ctx context.Context, query *entities.RQuery) (*entities.RQuery, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	ExistsActiveByIdentityOutsideProject(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter QueriesFilter) (int64, error)
}
