package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

func Query(value sqlc.Query) *entities.RQuery {
	return &entities.RQuery{
		ID:              value.ID,
		WorkspaceID:     value.WorkspaceID,
		ProjectID:       value.ProjectID,
		FolderID:        value.FolderID,
		Identity:        value.Identity,
		DisplayName:     value.DisplayName,
		Description:     NullableTextToEntity(value.Description),
		QueryType:       value.QueryType,
		Source:          JSONBToEntity(value.Source),
		Params:          JSONBArrayToEntity(value.Params),
		Headers:         JSONBToEntity(value.Headers),
		Auth:            NullableJSONBToEntity(value.Auth),
		TimeoutMS:       NullableInt4ToEntity(value.TimeoutMs),
		MockData:        NullableJSONBToEntity(value.MockData),
		MockDataEnabled: value.MockDataEnabled,
		Meta:            JSONBToEntity(value.Meta),
		Active:          value.Active,
		DeletedAt:       NullableTimeToEntity(value.DeletedAt),
		CreatedAt:       value.CreatedAt,
		UpdatedAt:       value.UpdatedAt,
	}
}

func CreateQueryParams(value *entities.RQuery) sqlc.CreateQueryParams {
	if value == nil {
		return sqlc.CreateQueryParams{}
	}
	return sqlc.CreateQueryParams{
		WorkspaceID:     value.WorkspaceID,
		ProjectID:       value.ProjectID,
		FolderID:        value.FolderID,
		Identity:        value.Identity,
		DisplayName:     value.DisplayName,
		Description:     NullableTextToSQLC(value.Description),
		QueryType:       value.QueryType,
		Source:          JSONBToSQLC(value.Source),
		Params:          JSONBArrayToSQLC(value.Params),
		Headers:         JSONBToSQLC(value.Headers),
		Auth:            NullableJSONBToSQLC(value.Auth),
		TimeoutMs:       NullableInt4ToSQLC(value.TimeoutMS),
		MockData:        NullableJSONBToSQLC(value.MockData),
		MockDataEnabled: value.MockDataEnabled,
		Meta:            JSONBToSQLC(value.Meta),
		Active:          value.Active,
	}
}

func UpdateQueryParams(value *entities.RQuery) sqlc.UpdateQueryParams {
	if value == nil {
		return sqlc.UpdateQueryParams{}
	}
	return sqlc.UpdateQueryParams{
		ID:              value.ID,
		WorkspaceID:     value.WorkspaceID,
		FolderID:        value.FolderID,
		DisplayName:     value.DisplayName,
		Description:     NullableTextToSQLC(value.Description),
		QueryType:       value.QueryType,
		Source:          JSONBToSQLC(value.Source),
		Params:          JSONBArrayToSQLC(value.Params),
		Headers:         JSONBToSQLC(value.Headers),
		Auth:            NullableJSONBToSQLC(value.Auth),
		TimeoutMs:       NullableInt4ToSQLC(value.TimeoutMS),
		MockData:        NullableJSONBToSQLC(value.MockData),
		MockDataEnabled: value.MockDataEnabled,
		Meta:            JSONBToSQLC(value.Meta),
		Active:          value.Active,
	}
}
