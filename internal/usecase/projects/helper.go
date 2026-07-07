package projects

import (
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/google/uuid"
)

func projectFromCreateInput(input adapters.CreateProjectInput) *entities.Project {
	return &entities.Project{
		Identity:              input.Identity,
		DisplayName:           input.DisplayName,
		ExtendSettings:        input.ExtendSettings,
		SettingsID:            input.SettingsID,
		NavigationID:          input.NavigationID,
		FolderID:              input.FolderID,
		AllowedEnvironmentIDs: input.AllowedEnvironmentIDs,
		Meta:                  input.Meta,
	}
}

func projectFromUpdateInput(input adapters.UpdateProjectInput) *entities.Project {
	return &entities.Project{
		ID:                    input.ID,
		Identity:              input.Identity,
		DisplayName:           input.DisplayName,
		ExtendSettings:        input.ExtendSettings,
		SettingsID:            input.SettingsID,
		NavigationID:          input.NavigationID,
		FolderID:              input.FolderID,
		AllowedEnvironmentIDs: input.AllowedEnvironmentIDs,
		Meta:                  input.Meta,
	}
}
func normalizeAndValidateUpdateProjectInput(input *adapters.UpdateProjectInput) error {
	if input.ID == uuid.Nil {
		return apperrors.InvalidInput("projects.empty_id", "project id is required")
	}

	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Identity == "" {
		return apperrors.InvalidInput("projects.empty_identity", "project identity is required")
	}

	if input.DisplayName == "" {
		return apperrors.InvalidInput("projects.empty_display_name", "project display name is required")
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	if input.AllowedEnvironmentIDs == nil {
		input.AllowedEnvironmentIDs = []uuid.UUID{}
	}

	return nil
}
