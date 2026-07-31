package mappers

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

// Tenant maps a persisted tenant and its JSONB contribution to the domain.
func Tenant(value sqlc.Tenant) (*entities.RTenant, error) {
	var configuration entities.EndgeConfigurationContribution
	if err := json.Unmarshal(value.Configuration, &configuration); err != nil {
		return nil, err
	}

	return &entities.RTenant{
		ID:            value.ID,
		WorkspaceID:   value.WorkspaceID,
		Identity:      value.Identity,
		DisplayName:   value.DisplayName,
		Code:          value.Code,
		Description:   NullableTextToEntity(value.Description),
		FolderID:      NullableUUIDToEntity(value.FolderID),
		Configuration: configuration,
		CreatedAt:     value.CreatedAt,
		UpdatedAt:     value.UpdatedAt,
	}, nil
}

// CreateTenantParams serializes the tenant contribution for JSONB storage.
func CreateTenantParams(value *entities.RTenant) (sqlc.CreateTenantParams, error) {
	if value == nil {
		return sqlc.CreateTenantParams{}, nil
	}

	configuration, err := json.Marshal(value.Configuration)
	if err != nil {
		return sqlc.CreateTenantParams{}, err
	}

	return sqlc.CreateTenantParams{
		WorkspaceID:   value.WorkspaceID,
		Identity:      value.Identity,
		DisplayName:   value.DisplayName,
		Code:          value.Code,
		Description:   NullableTextToSQLC(value.Description),
		FolderID:      NullableUUIDToSQLC(value.FolderID),
		Configuration: configuration,
	}, nil
}

// UpdateTenantParams serializes a complete tenant persistence update.
func UpdateTenantParams(value *entities.RTenant) (sqlc.UpdateTenantParams, error) {
	if value == nil {
		return sqlc.UpdateTenantParams{}, nil
	}

	configuration, err := json.Marshal(value.Configuration)
	if err != nil {
		return sqlc.UpdateTenantParams{}, err
	}

	return sqlc.UpdateTenantParams{
		WorkspaceID:   value.WorkspaceID,
		Identity:      value.Identity,
		DisplayName:   value.DisplayName,
		Code:          value.Code,
		Description:   NullableTextToSQLC(value.Description),
		FolderID:      NullableUUIDToSQLC(value.FolderID),
		Configuration: configuration,
	}, nil
}
