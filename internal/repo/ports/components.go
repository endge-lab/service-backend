package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type ComponentsFilter struct {
	ProjectID     uuid.UUID
	FolderID      *uuid.UUID
	ComponentType *entities.ComponentType
}

type ComponentsRepository interface {
	Create(
		ctx context.Context,
		component *entities.Component,
	) (componentResult *entities.Component, err error)

	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (componentResult *entities.Component, err error)

	GetByIdentity(
		ctx context.Context,
		projectID uuid.UUID,
		identity string,
	) (componentResult *entities.Component, err error)

	GetByIdentityIncludingDeleted(
		ctx context.Context,
		projectID uuid.UUID,
		identity string,
	) (componentResult *entities.Component, err error)

	List(
		ctx context.Context,
		filter ComponentsFilter,
	) (components []*entities.Component, err error)

	Update(
		ctx context.Context,
		component *entities.Component,
	) (componentResult *entities.Component, err error)

	SoftDelete(
		ctx context.Context,
		id uuid.UUID,
	) (err error)

	Restore(
		ctx context.Context,
		id uuid.UUID,
	) (err error)

	HardDelete(
		ctx context.Context,
		id uuid.UUID,
	) (err error)

	ExistsByIdentity(
		ctx context.Context,
		projectID uuid.UUID,
		identity string,
	) (exists bool, err error)

	Count(
		ctx context.Context,
		filter ComponentsFilter,
	) (count int64, err error)
}
