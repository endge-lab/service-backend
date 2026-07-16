package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func Folder(value sqlc.Folder) *entities.RFolder {
	return &entities.RFolder{ID: value.ID, ProjectID: NullableUUIDToEntity(value.ProjectID), EntityType: entities.FolderEntityType(value.EntityType), Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToEntity(value.Description), ParentID: NullableUUIDToEntity(value.ParentID), IsRoot: value.IsRoot, IsSystem: value.IsSystem, DeletedAt: NullableTimeToEntity(value.DeletedAt), Meta: JSONBToEntity(value.Meta), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func CreateFolderParams(value *entities.RFolder) sqlc.CreateFolderParams {
	if value == nil {
		return sqlc.CreateFolderParams{}
	}
	return sqlc.CreateFolderParams{ProjectID: NullableUUIDToSQLC(value.ProjectID), EntityType: string(value.EntityType), Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ParentID: NullableUUIDToSQLC(value.ParentID), IsRoot: value.IsRoot, IsSystem: value.IsSystem, Meta: JSONBToSQLC(value.Meta)}
}
func UpdateFolderParams(value *entities.RFolder) sqlc.UpdateFolderParams {
	if value == nil {
		return sqlc.UpdateFolderParams{}
	}
	return sqlc.UpdateFolderParams{ID: value.ID, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ParentID: NullableUUIDToSQLC(value.ParentID), Meta: JSONBToSQLC(value.Meta)}
}
