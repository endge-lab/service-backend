package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

// DomainDependenciesListOptions задаёт ограниченную страницу derived usages.
type DomainDependenciesListOptions struct {
	Limit  int
	Offset int
}

// DomainDependenciesRepository сохраняет derived dependency projection
// workspace-scoped canonical documents. Workspace boundary всегда читается из
// ctx: caller не передаёт его как недоверенное transport-значение.
type DomainDependenciesRepository interface {
	ReplaceForOwner(ctx context.Context, owner entities.DomainDependencyOwner, references []entities.DomainDependencyReference, state entities.DomainDependencyVerificationState, verificationError *string) error
	DeleteForOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error
	ListUsages(ctx context.Context, dependencyType, dependencyIdentity string, options DomainDependenciesListOptions) (entities.DomainDependencyUsages, error)
	EnsureNotReferenced(ctx context.Context, dependencyType, dependencyIdentity string, limit int) (entities.DomainDependencyUsages, error)
}
