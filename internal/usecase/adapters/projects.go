package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type ProjectService interface {
	Create(ctx context.Context, input CreateProjectInput) (*entities.Project, error)
	Update(ctx context.Context, input UpdateProjectInput) (*entities.Project, error)

	GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.Project, error)
	List(ctx context.Context) ([]*entities.Project, error)

	SoftDelete(ctx context.Context, identity string) error
	Restore(ctx context.Context, identity string) error
	HardDelete(ctx context.Context, identity string) error
	Count(ctx context.Context) (int64, error)
}

type CreateProjectInput struct {
	Identity    string
	DisplayName string
	Description *string
	Active      bool
	Meta        map[string]any
}

type UpdateProjectInput struct {
	Identity    string
	DisplayName string
	Description *string
	Active      bool
	Meta        map[string]any
}
