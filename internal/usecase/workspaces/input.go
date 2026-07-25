package workspaces

import "github.com/endge-lab/service-backend/internal/domain/entities"

type CreateWorkspaceInput struct {
	Identity      string
	DisplayName   string
	Configuration *entities.EndgeConfiguration
}

type UpdateWorkspaceInput struct {
	Identity      string
	DisplayName   *string
	Configuration *entities.EndgeConfiguration
}
