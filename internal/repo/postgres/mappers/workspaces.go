package mappers

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

// Workspace converts the persisted JSONB configuration into the complete domain model.
func Workspace(value sqlc.Workspace) (*entities.RWorkspace, error) {
	var configuration entities.EndgeConfiguration
	if err := json.Unmarshal(value.Configuration, &configuration); err != nil {
		return nil, err
	}

	return &entities.RWorkspace{
		ID:            value.ID,
		Identity:      value.Identity,
		DisplayName:   value.DisplayName,
		Configuration: configuration,
		CreatedAt:     value.CreatedAt,
		UpdatedAt:     value.UpdatedAt,
	}, nil
}

// CreateWorkspaceParams serializes the complete root configuration for insertion.
func CreateWorkspaceParams(value *entities.RWorkspace) (sqlc.CreateWorkspaceParams, error) {
	if value == nil {
		return sqlc.CreateWorkspaceParams{}, nil
	}

	configuration, err := json.Marshal(value.Configuration)
	if err != nil {
		return sqlc.CreateWorkspaceParams{}, err
	}

	return sqlc.CreateWorkspaceParams{
		Identity:      value.Identity,
		DisplayName:   value.DisplayName,
		Configuration: configuration,
	}, nil
}

// UpdateWorkspaceParams serializes the full replacement configuration for update.
func UpdateWorkspaceParams(value *entities.RWorkspace) (sqlc.UpdateWorkspaceParams, error) {
	if value == nil {
		return sqlc.UpdateWorkspaceParams{}, nil
	}

	configuration, err := json.Marshal(value.Configuration)
	if err != nil {
		return sqlc.UpdateWorkspaceParams{}, err
	}

	return sqlc.UpdateWorkspaceParams{
		ID:            value.ID,
		DisplayName:   value.DisplayName,
		Configuration: configuration,
	}, nil
}
