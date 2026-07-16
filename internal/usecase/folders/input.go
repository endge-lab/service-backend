package folders

import "github.com/endge-lab/service-backend/internal/domain/entities"

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
