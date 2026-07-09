package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type FolderService interface {
	Create(ctx context.Context, input CreateFolderInput) (*entities.Folder, error)
	Update(ctx context.Context, input UpdateFolderInput) (*entities.Folder, error)

	GetByID(ctx context.Context, id uuid.UUID) (*entities.Folder, error)
	GetByIdentity(ctx context.Context, input GetFolderInput) (*entities.Folder, error)
	List(ctx context.Context, input ListFoldersInput) ([]*entities.Folder, error)

	SoftDelete(ctx context.Context, input FolderIdentityInput) error
	Restore(ctx context.Context, input FolderIdentityInput) error
	HardDelete(ctx context.Context, input FolderIdentityInput) error
	Count(ctx context.Context, input ListFoldersInput) (int64, error)
}

type CreateFolderInput struct {
	ProjectIdentity string
	EntityType      entities.FolderEntityType
	Identity        string
	DisplayName     string
	Description     *string
	ParentIdentity  *string
	Meta            map[string]any
}

type UpdateFolderInput struct {
	ProjectIdentity string
	EntityType      entities.FolderEntityType
	Identity        string
	DisplayName     string
	Description     *string
	ParentIdentity  *string
	Meta            map[string]any
}

type GetFolderInput struct {
	ProjectIdentity string
	EntityType      entities.FolderEntityType
	Identity        string
}

type ListFoldersInput struct {
	ProjectIdentity string
	EntityType      entities.FolderEntityType
}

type FolderIdentityInput struct {
	ProjectIdentity string
	EntityType      entities.FolderEntityType
	Identity        string
}
