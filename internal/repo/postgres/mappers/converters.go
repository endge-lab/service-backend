package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func Converter(value sqlc.Converter) *entities.RConverter {
	return &entities.RConverter{ID: value.ID, WorkspaceID: value.WorkspaceID, ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToEntity(value.Description), ConverterType: value.ConverterType, Source: JSONBToEntity(value.Source), IsSystem: value.IsSystem, Meta: JSONBToEntity(value.Meta), Active: value.Active, DeletedAt: NullableTimeToEntity(value.DeletedAt), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func CreateConverterParams(value *entities.RConverter) sqlc.CreateConverterParams {
	if value == nil {
		return sqlc.CreateConverterParams{}
	}
	return sqlc.CreateConverterParams{WorkspaceID: value.WorkspaceID, ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ConverterType: value.ConverterType, Source: JSONBToSQLC(value.Source), IsSystem: value.IsSystem, Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
func UpdateConverterParams(value *entities.RConverter) sqlc.UpdateConverterParams {
	if value == nil {
		return sqlc.UpdateConverterParams{}
	}
	return sqlc.UpdateConverterParams{ID: value.ID, WorkspaceID: value.WorkspaceID, FolderID: value.FolderID, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ConverterType: value.ConverterType, Source: JSONBToSQLC(value.Source), IsSystem: value.IsSystem, Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
