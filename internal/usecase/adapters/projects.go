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

	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

type CreateProjectInput struct {
	Identity              string
	DisplayName           string
	ExtendSettings        bool
	SettingsID            *uuid.UUID
	NavigationID          *uuid.UUID
	FolderID              *uuid.UUID
	AllowedEnvironmentIDs []uuid.UUID
	Meta                  map[string]any
}

type UpdateProjectInput struct {
	ID                    uuid.UUID
	Identity              string
	DisplayName           string
	ExtendSettings        bool
	SettingsID            *uuid.UUID
	NavigationID          *uuid.UUID
	FolderID              *uuid.UUID
	AllowedEnvironmentIDs []uuid.UUID
	Meta                  map[string]any
}
