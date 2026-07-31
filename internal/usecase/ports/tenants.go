package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

// TenantsFilter narrows a workspace-scoped tenant list.
type TenantsFilter struct {
	FolderID *uuid.UUID
}

// TenantsRepository persists the final tenant configuration layer.
// The workspace boundary is always taken from ctx.
type TenantsRepository interface {
	Create(ctx context.Context, tenant *entities.RTenant) (*entities.RTenant, error)
	List(ctx context.Context, filter TenantsFilter) ([]*entities.RTenant, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.RTenant, error)
	Update(ctx context.Context, tenant *entities.RTenant) (*entities.RTenant, error)
	HardDelete(ctx context.Context, identity string) error
}
