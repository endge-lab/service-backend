package projects

import (
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/google/uuid"
)

var projectRootEntityTypes = []entities.FolderEntityType{
	entities.FolderEntityTypeComponentsLegacy,
	entities.FolderEntityTypeConverters,
	entities.FolderEntityTypeQueries,
	entities.FolderEntityTypeDataViews,
}

func projectFromCreateInput(input CreateProjectInput) *entities.RProject {
	return &entities.RProject{
		Identity:    input.Identity,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Active:      input.Active,
		Meta:        input.Meta,
	}
}

func projectFromUpdateInput(current *entities.RProject, input UpdateProjectInput) *entities.RProject {
	return &entities.RProject{
		ID:          current.ID,
		Identity:    current.Identity,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Active:      input.Active,
		DeletedAt:   current.DeletedAt,
		Meta:        input.Meta,
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   current.UpdatedAt,
	}
}

func projectRootFolders(projectID uuid.UUID) []*entities.RFolder {
	roots := make([]*entities.RFolder, 0, len(projectRootEntityTypes))
	for _, entityType := range projectRootEntityTypes {
		id := projectID
		identity := "root-" + string(entityType)
		roots = append(roots, &entities.RFolder{
			ProjectID:   &id,
			EntityType:  entityType,
			Identity:    identity,
			DisplayName: identity,
			IsRoot:      true,
			IsSystem:    true,
			Meta:        map[string]any{},
		})
	}

	return roots
}

func normalizeAndValidateUpdateProjectInput(input *UpdateProjectInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Identity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}

	if input.DisplayName == "" {
		return apperrors.InvalidInput("validation_error", "project display name is required")
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	return nil
}
