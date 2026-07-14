package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func Component(value sqlc.Component) *entities.Component {
	return &entities.Component{ID: value.ID, ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToEntity(value.Description), ComponentType: entities.ComponentType(value.ComponentType), Source: value.Source, SourceFormat: entities.ComponentSourceFormat(value.SourceFormat), PropsSchema: JSONBToEntity(value.PropsSchema), Bindings: JSONBToEntity(value.Bindings), Meta: JSONBToEntity(value.Meta), Active: value.Active, DeletedAt: NullableTimeToEntity(value.DeletedAt), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func CreateComponentParams(value *entities.Component) sqlc.CreateComponentParams {
	if value == nil {
		return sqlc.CreateComponentParams{}
	}
	return sqlc.CreateComponentParams{ProjectID: value.ProjectID, FolderID: value.FolderID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ComponentType: string(value.ComponentType), Source: value.Source, SourceFormat: string(value.SourceFormat), PropsSchema: JSONBToSQLC(value.PropsSchema), Bindings: JSONBToSQLC(value.Bindings), Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
func UpdateComponentParams(value *entities.Component) sqlc.UpdateComponentParams {
	if value == nil {
		return sqlc.UpdateComponentParams{}
	}
	return sqlc.UpdateComponentParams{ID: value.ID, FolderID: value.FolderID, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), ComponentType: string(value.ComponentType), Source: value.Source, SourceFormat: string(value.SourceFormat), PropsSchema: JSONBToSQLC(value.PropsSchema), Bindings: JSONBToSQLC(value.Bindings), Meta: JSONBToSQLC(value.Meta), Active: value.Active}
}
