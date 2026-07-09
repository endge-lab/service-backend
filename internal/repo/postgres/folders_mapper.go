package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func mapFolder(folder sqlc.Folder) *entities.Folder {
	return &entities.Folder{
		ID:          folder.ID,
		ProjectID:   mapNullableUUIDToEntity(folder.ProjectID),
		EntityType:  entities.FolderEntityType(folder.EntityType),
		Identity:    folder.Identity,
		DisplayName: folder.DisplayName,
		Description: mapNullableTextToEntity(folder.Description),
		ParentID:    mapNullableUUIDToEntity(folder.ParentID),
		IsRoot:      folder.IsRoot,
		IsSystem:    folder.IsSystem,
		DeletedAt:   mapNullableTimeToEntity(folder.DeletedAt),
		Meta:        mapJSONBToEntity(folder.Meta),
		CreatedAt:   folder.CreatedAt,
		UpdatedAt:   folder.UpdatedAt,
	}
}

func mapCreateFolderParams(folder *entities.Folder) sqlc.CreateFolderParams {
	if folder == nil {
		return sqlc.CreateFolderParams{}
	}

	return sqlc.CreateFolderParams{
		ProjectID:   mapNullableUUIDToSQLC(folder.ProjectID),
		EntityType:  string(folder.EntityType),
		Identity:    folder.Identity,
		DisplayName: folder.DisplayName,
		Description: mapNullableTextToSQLC(folder.Description),
		ParentID:    mapNullableUUIDToSQLC(folder.ParentID),
		IsRoot:      folder.IsRoot,
		IsSystem:    folder.IsSystem,
		Meta:        mapJSONBToSQLC(folder.Meta),
	}
}

func mapUpdateFolderParams(folder *entities.Folder) sqlc.UpdateFolderParams {
	if folder == nil {
		return sqlc.UpdateFolderParams{}
	}

	return sqlc.UpdateFolderParams{
		ID:          folder.ID,
		DisplayName: folder.DisplayName,
		Description: mapNullableTextToSQLC(folder.Description),
		ParentID:    mapNullableUUIDToSQLC(folder.ParentID),
		Meta:        mapJSONBToSQLC(folder.Meta),
	}
}
