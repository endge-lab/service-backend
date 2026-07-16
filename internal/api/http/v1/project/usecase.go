package project

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
)

// UseCase is the application contract consumed by the project HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input projects.CreateProjectInput) (*entities.RProject, error)
	Update(ctx context.Context, input projects.UpdateProjectInput) (*entities.RProject, error)
	GetByIdentity(ctx context.Context, identity string) (*entities.RProject, error)
	List(ctx context.Context) ([]*entities.RProject, error)
	SoftDelete(ctx context.Context, identity string) error
	Restore(ctx context.Context, identity string) error
	HardDelete(ctx context.Context, identity string) error
}
