package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

// ProjectsRepository defines persistence operations required by project use cases.
type ProjectsRepository interface {
	Create(ctx context.Context, project *entities.RProject) (*entities.RProject, error)

	GetByID(ctx context.Context, id uuid.UUID) (*entities.RProject, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.RProject, error)
	GetByIdentityIncludingDeleted(ctx context.Context, identity string) (*entities.RProject, error)

	List(ctx context.Context) ([]*entities.RProject, error)

	Update(ctx context.Context, project *entities.RProject) (*entities.RProject, error)

	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error

	ExistsByIdentity(ctx context.Context, identity string) (bool, error)
	Count(ctx context.Context) (int64, error)
}
