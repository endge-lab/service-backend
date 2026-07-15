package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type FoldersRepository interface {
	Create(ctx context.Context, folder *entities.Folder) (*entities.Folder, error)
	Update(ctx context.Context, folder *entities.Folder) (*entities.Folder, error)

	GetByID(ctx context.Context, id uuid.UUID) (*entities.Folder, error)
	GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entities.Folder, error)
	GetByIdentity(
		ctx context.Context,
		projectID *uuid.UUID,
		entityType entities.FolderEntityType,
		identity string,
	) (*entities.Folder, error)
	GetByIdentityIncludingDeleted(
		ctx context.Context,
		projectID *uuid.UUID,
		entityType entities.FolderEntityType,
		identity string,
	) (*entities.Folder, error)
	List(
		ctx context.Context,
		projectID *uuid.UUID,
		entityType entities.FolderEntityType,
	) ([]*entities.Folder, error)

	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error

	ExistsByIdentity(
		ctx context.Context,
		projectID *uuid.UUID,
		entityType entities.FolderEntityType,
		identity string,
	) (bool, error)
	Count(
		ctx context.Context,
		projectID *uuid.UUID,
		entityType entities.FolderEntityType,
	) (int64, error)
}
