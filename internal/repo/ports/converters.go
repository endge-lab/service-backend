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
	Create(ctx context.Context, converter *entities.Converter) (*entities.Converter, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Converter, error)
	GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Converter, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Converter, error)
	List(ctx context.Context, filter ConvertersFilter) ([]*entities.Converter, error)
	Update(ctx context.Context, converter *entities.Converter) (*entities.Converter, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (bool, error)
	Count(ctx context.Context, filter ConvertersFilter) (int64, error)
}
