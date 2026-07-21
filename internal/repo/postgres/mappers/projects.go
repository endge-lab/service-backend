package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func Project(value sqlc.Project) *entities.RProject {
	return &entities.RProject{ID: value.ID, WorkspaceID: value.WorkspaceID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToEntity(value.Description), Active: value.Active, DeletedAt: NullableTimeToEntity(value.DeletedAt), Meta: JSONBToEntity(value.Meta), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func CreateProjectParams(value *entities.RProject) sqlc.CreateProjectParams {
	if value == nil {
		return sqlc.CreateProjectParams{}
	}
	return sqlc.CreateProjectParams{WorkspaceID: value.WorkspaceID, Identity: value.Identity, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), Active: value.Active, Meta: JSONBToSQLC(value.Meta)}
}
func UpdateProjectParams(value *entities.RProject) sqlc.UpdateProjectParams {
	if value == nil {
		return sqlc.UpdateProjectParams{}
	}
	return sqlc.UpdateProjectParams{ID: value.ID, WorkspaceID: value.WorkspaceID, DisplayName: value.DisplayName, Description: NullableTextToSQLC(value.Description), Active: value.Active, Meta: JSONBToSQLC(value.Meta)}
}
