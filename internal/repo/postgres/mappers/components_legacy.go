package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func ComponentLegacy(value sqlc.ComponentLegacy) *entities.RComponentLegacy {
	return &entities.RComponentLegacy{ID: value.ID, ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToEntity(value.Description), ComponentType: entities.RComponentLegacyType(value.ComponentType), Source: value.Source, SourceFormat: entities.RComponentLegacySourceFormat(value.SourceFormat), PropsSchema: JSONBToEntity(value.PropsSchema), Bindings: JSONBToEntity(value.Bindings), Meta: JSONBToEntity(value.Meta), Active: value.Active, DeletedAt: NullableTimeToEntity(value.DeletedAt), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func CreateComponentLegacyParams(value *entities.RComponentLegacy) sqlc.CreateComponentLegacyParams {
	if value == nil {
		return sqlc.CreateComponentLegacyParams{}
	}
	return sqlc.CreateComponentLegacyParams{ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ComponentType: string(value.ComponentType), Source: value.Source, SourceFormat: string(value.SourceFormat), PropsSchema: JSONBToSQLC(value.PropsSchema), Bindings: JSONBToSQLC(value.Bindings), Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
func UpdateComponentLegacyParams(value *entities.RComponentLegacy) sqlc.UpdateComponentLegacyParams {
	if value == nil {
		return sqlc.UpdateComponentLegacyParams{}
	}
	return sqlc.UpdateComponentLegacyParams{ID: value.ID, FolderID: value.FolderID, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ComponentType: string(value.ComponentType), Source: value.Source, SourceFormat: string(value.SourceFormat), PropsSchema: JSONBToSQLC(value.PropsSchema), Bindings: JSONBToSQLC(value.Bindings), Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
