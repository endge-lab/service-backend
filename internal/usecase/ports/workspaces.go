package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// WorkspacesRepository defines persistence operations required by workspace use cases.
type WorkspacesRepository interface {
	Create(ctx context.Context, workspace *entities.RWorkspace) (*entities.RWorkspace, error)
	List(ctx context.Context) ([]*entities.RWorkspace, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.RWorkspace, error)
	Update(ctx context.Context, workspace *entities.RWorkspace) (*entities.RWorkspace, error)
}
