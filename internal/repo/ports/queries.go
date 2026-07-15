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

type QueriesRepository interface {
	Create(ctx context.Context, query *entities.Query) (*entities.Query, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Query, error)
	GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entities.Query, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Query, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Query, error)
	List(ctx context.Context, filter QueriesFilter) ([]*entities.Query, error)
	Update(ctx context.Context, query *entities.Query) (*entities.Query, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	ExistsActiveByIdentityOutsideProject(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter QueriesFilter) (int64, error)
}
