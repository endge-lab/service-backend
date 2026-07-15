package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"

	"github.com/google/uuid"
)

type ConvertersFilter struct {
	ProjectID uuid.UUID
	FolderID  *uuid.UUID
}

type ConvertersRepository interface {
	Create(
		ctx context.Context,
		converter *entities.Converter,
	) (converterResult *entities.Converter, err error)

	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (converterResult *entities.Converter, err error)

	GetByIdentity(
		ctx context.Context,
		projectID uuid.UUID,
		identity string,
	) (converterResult *entities.Converter, err error)

	GetByIdentityIncludingDeleted(
		ctx context.Context,
		projectID uuid.UUID,
		identity string,
	) (converterResult *entities.Converter, err error)

	List(
		ctx context.Context,
		filter ConvertersFilter,
	) (converters []*entities.Converter, err error)

	Update(
		ctx context.Context,
		converter *entities.Converter,
	) (converterResult *entities.Converter, err error)

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
		filter ConvertersFilter,
	) (count int64, err error)
}
