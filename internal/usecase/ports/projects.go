package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type ProjectsRepository interface {
	Create(ctx context.Context, project *entities.Project) (*entities.Project, error)

	GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.Project, error)

	List(ctx context.Context) ([]*entities.Project, error)

	Update(ctx context.Context, project *entities.Project) (*entities.Project, error)

	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error

	ExistsByIdentity(ctx context.Context, identity string) (bool, error)
	Count(ctx context.Context) (int64, error)
}
