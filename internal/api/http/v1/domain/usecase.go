package domain

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/dependencies"
)

// UseCase — application contract, который использует HTTP adapter domain.
type UseCase interface {
	ListUsages(context.Context, dependencies.ListUsagesInput) (entities.DomainDependencyUsages, error)
}
