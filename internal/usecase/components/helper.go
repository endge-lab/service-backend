package components

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
)

func (c *Component) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}
	folder, err := c.folderRepository.GetByIdentity(ctx, &projectID, entities.FolderEntityTypeComponents, *identity)
	if err != nil {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have components entity type")
	}
	return &folder.ID, nil
}

func (c *Component) resolveComponent(ctx context.Context, input adapters.ComponentIdentityInput, includeDeleted bool) (*entities.Component, error) {
	if err := normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.ComponentIdentity); err != nil {
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	if includeDeleted {
		return c.componentRepository.GetByIdentityIncludingDeleted(ctx, project.ID, input.ComponentIdentity)
	}
	return c.componentRepository.GetByIdentity(ctx, project.ID, input.ComponentIdentity)
}

func normalizeAndValidateCreateInput(input *adapters.CreateComponentInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.FolderIdentity = strings.TrimSpace(input.FolderIdentity)
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Source = strings.TrimSpace(input.Source)

	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		input.Description = &description
	}

	if err := validateComponentFields(
		input.ProjectIdentity,
		input.FolderIdentity,
		input.ComponentType,
		input.Identity,
		input.DisplayName,
		input.Source,
	); err != nil {
		return err
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	if input.Bindings == nil {
		input.Bindings = map[string]any{}
	}

	if input.PropsSchema == nil {
		input.PropsSchema = map[string]any{}
	}

	return nil
}

func normalizeAndValidateUpdateInput(input *adapters.UpdateComponentInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.FolderIdentity = strings.TrimSpace(input.FolderIdentity)
	input.ComponentIdentity = strings.TrimSpace(input.ComponentIdentity)

	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Source = strings.TrimSpace(input.Source)

	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		input.Description = &description
	}

	if err := validateComponentFields(
		input.ProjectIdentity,
		input.FolderIdentity,
		input.ComponentType,
		input.ComponentIdentity,
		input.DisplayName,
		input.Source,
	); err != nil {
		return err
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	if input.Bindings == nil {
		input.Bindings = map[string]any{}
	}

	if input.PropsSchema == nil {
		input.PropsSchema = map[string]any{}
	}

	return nil
}

func validateComponentFields(
	projectIdentity string,
	folderIdentity string,
	componentType entities.ComponentType,
	identity string,
	displayName string,
	source string,
) error {
	if projectIdentity == "" ||
		folderIdentity == "" ||
		identity == "" ||
		displayName == "" ||
		source == "" {
		return apperrors.InvalidInput(
			"validation_error",
			"project identity, folder identity, component identity, display name and source are required",
		)
	}

	if !isSupportedComponentType(componentType) {
		return apperrors.InvalidInput(
			"validation_error",
			"unsupported component type",
		)
	}

	return nil
}

func isSupportedComponentType(componentType entities.ComponentType) bool {
	return componentType == entities.ComponentTypeSFC
}

func normalizeAndValidateIdentityInput(
	projectIdentity *string,
	componentIdentity *string,
) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*componentIdentity = strings.TrimSpace(*componentIdentity)

	if *projectIdentity == "" || *componentIdentity == "" {
		return apperrors.InvalidInput(
			"validation_error",
			"project identity and component identity are required",
		)
	}

	return nil
}

func normalizeAndValidateListInput(
	projectIdentity *string,
	folderIdentity *string,
	componentType *entities.ComponentType,
) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)

	if *projectIdentity == "" {
		return apperrors.InvalidInput(
			"validation_error",
			"project identity is required",
		)
	}

	if folderIdentity != nil {
		folder := strings.TrimSpace(*folderIdentity)
		*folderIdentity = folder
	}

	if componentType != nil && !isSupportedComponentType(*componentType) {
		return apperrors.InvalidInput(
			"validation_error",
			"unsupported component type",
		)
	}

	return nil
}
