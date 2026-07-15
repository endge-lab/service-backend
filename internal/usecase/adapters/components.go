package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateComponentInput struct {
	ProjectIdentity string
	FolderIdentity  string

	Identity      string
	DisplayName   string
	Description   *string
	ComponentType entities.ComponentType
	Source        string

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any
	Active      bool
}

type UpdateComponentInput struct {
	ProjectIdentity   string
	ComponentIdentity string
	FolderIdentity    string

	DisplayName   string
	Description   *string
	ComponentType entities.ComponentType
	Source        string

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any
	Active      bool
}

type GetComponentInput struct {
	ProjectIdentity   string
	ComponentIdentity string
}

type ComponentIdentityInput struct {
	ProjectIdentity   string
	ComponentIdentity string
}

type ListComponentsInput struct {
	ProjectIdentity string
	FolderIdentity  *string
	ComponentType   *entities.ComponentType
}

type ComponentWithFolder struct {
	Component      *entities.Component
	FolderIdentity string
}

type ComponentService interface {
	Create(ctx context.Context, input CreateComponentInput) (*ComponentWithFolder, error)

	Update(ctx context.Context, input UpdateComponentInput) (*ComponentWithFolder, error)

	GetByIdentity(ctx context.Context, input GetComponentInput) (*ComponentWithFolder, error)

	List(ctx context.Context, input ListComponentsInput) ([]*ComponentWithFolder, error)

	SoftDelete(ctx context.Context, input ComponentIdentityInput) error

	Restore(ctx context.Context, input ComponentIdentityInput) error

	HardDelete(ctx context.Context, input ComponentIdentityInput) error

	Count(ctx context.Context, input ListComponentsInput) (int64, error)
}
