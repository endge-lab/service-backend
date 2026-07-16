package folder

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/google/uuid"
)

// UseCase is the application contract consumed by the folder HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input folders.CreateFolderInput) (*entities.RFolder, error)
	Update(ctx context.Context, input folders.UpdateFolderInput) (*entities.RFolder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RFolder, error)
	GetByIdentity(ctx context.Context, input folders.GetFolderInput) (*entities.RFolder, error)
	List(ctx context.Context, input folders.ListFoldersInput) ([]*entities.RFolder, error)
	SoftDelete(ctx context.Context, input folders.FolderIdentityInput) error
	Restore(ctx context.Context, input folders.FolderIdentityInput) error
	HardDelete(ctx context.Context, input folders.FolderIdentityInput) error
}
