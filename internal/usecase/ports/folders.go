package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

// FoldersRepository defines persistence operations required by folder use cases.
type FoldersRepository interface {
	Create(ctx context.Context, folder *entities.RFolder) (*entities.RFolder, error)
	Update(ctx context.Context, folder *entities.RFolder) (*entities.RFolder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RFolder, error)
	GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entities.RFolder, error)
	GetByIdentity(ctx context.Context, projectID *uuid.UUID, entityType entities.FolderEntityType, identity string) (*entities.RFolder, error)
	GetByIdentityIncludingDeleted(ctx context.Context, projectID *uuid.UUID, entityType entities.FolderEntityType, identity string) (*entities.RFolder, error)
	List(ctx context.Context, projectID *uuid.UUID, entityType entities.FolderEntityType) ([]*entities.RFolder, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ExistsByIdentity(ctx context.Context, projectID *uuid.UUID, entityType entities.FolderEntityType, identity string) (bool, error)
	Count(ctx context.Context, projectID *uuid.UUID, entityType entities.FolderEntityType) (int64, error)
}
