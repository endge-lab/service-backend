package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func DataView(value sqlc.DataView) *entities.RDataView {
	return &entities.RDataView{
		ID:           value.ID,
		WorkspaceID:  value.WorkspaceID,
		ProjectID:    value.ProjectID,
		FolderID:     value.FolderID,
		QueryID:      value.QueryID,
		Identity:     value.Identity,
		DisplayName:  value.DisplayName,
		Description:  NullableTextToEntity(value.Description),
		ViewType:     value.ViewType,
		Source:       JSONBToEntity(value.Source),
		InputSchema:  JSONBToEntity(value.InputSchema),
		OutputSchema: JSONBToEntity(value.OutputSchema),
		Meta:         JSONBToEntity(value.Meta),
		Active:       value.Active,
		DeletedAt:    NullableTimeToEntity(value.DeletedAt),
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}
}

func CreateDataViewParams(value *entities.RDataView) sqlc.CreateDataViewParams {
	if value == nil {
		return sqlc.CreateDataViewParams{}
	}
	return sqlc.CreateDataViewParams{
		WorkspaceID:  value.WorkspaceID,
		ProjectID:    value.ProjectID,
		FolderID:     value.FolderID,
		QueryID:      value.QueryID,
		Identity:     value.Identity,
		DisplayName:  value.DisplayName,
		Description:  NullableTextToSQLC(value.Description),
		ViewType:     value.ViewType,
		Source:       JSONBToSQLC(value.Source),
		InputSchema:  JSONBToSQLC(value.InputSchema),
		OutputSchema: JSONBToSQLC(value.OutputSchema),
		Meta:         JSONBToSQLC(value.Meta),
		Active:       value.Active,
	}
}

func UpdateDataViewParams(value *entities.RDataView) sqlc.UpdateDataViewParams {
	if value == nil {
		return sqlc.UpdateDataViewParams{}
	}
	return sqlc.UpdateDataViewParams{
		ID:           value.ID,
		WorkspaceID:  value.WorkspaceID,
		FolderID:     value.FolderID,
		QueryID:      value.QueryID,
		DisplayName:  value.DisplayName,
		Description:  NullableTextToSQLC(value.Description),
		ViewType:     value.ViewType,
		Source:       JSONBToSQLC(value.Source),
		InputSchema:  JSONBToSQLC(value.InputSchema),
		OutputSchema: JSONBToSQLC(value.OutputSchema),
		Meta:         JSONBToSQLC(value.Meta),
		Active:       value.Active,
	}
}
