package workspaces

import (
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
)

func normalizeCreateInput(input *CreateWorkspaceInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Identity == "" || input.DisplayName == "" {
		return apperrors.InvalidInput("validation_error", "workspace identity and display name are required")
	}
	return nil
}

func normalizeUpdateInput(input *UpdateWorkspaceInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	if input.Identity == "" {
		return apperrors.InvalidInput("validation_error", "workspace identity is required")
	}
	if input.DisplayName != nil {
		value := strings.TrimSpace(*input.DisplayName)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "workspace display name is required")
		}
		input.DisplayName = &value
	}
	if input.DisplayName == nil && input.Configuration == nil {
		return apperrors.InvalidInput("validation_error", "workspace update is empty")
	}
	return nil
}

func validateConfiguration(c entities.EndgeConfiguration) error {
	return shared.ValidateEndgeConfiguration(c)
}
