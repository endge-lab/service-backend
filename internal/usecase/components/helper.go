package components

import (
	"context"
	"errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func logOperationError(logger *zap.Logger, operation string, err error, fields ...zap.Field) {
	fields = append([]zap.Field{zap.Error(err)}, fields...)
	if apperrors.HTTPStatusOf(err) >= 500 {
		logger.Error(operation, fields...)
		return
	}

	logger.Warn(operation, fields...)
}

func dereferenceString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func (c *Component) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}

	folder, err := c.resolveFolder(ctx, projectID, *identity)
	if err != nil {
		return nil, err
	}

	return &folder.ID, nil
}

func (c *Component) resolveFolder(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Folder, error) {
	folder, err := c.folderRepository.GetByIdentity(ctx, &projectID, entities.FolderEntityTypeComponents, identity)
	if err == nil {
		return folder, nil
	}
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.InvalidInput(
			"folder_entity_type_mismatch",
			"folder must belong to the project and have components entity type",
		)
	}

	return nil, err
}

func componentWithFolder(component *entities.Component, folderIdentity string) *adapters.ComponentWithFolder {
	return &adapters.ComponentWithFolder{
		Component:      component,
		FolderIdentity: folderIdentity,
	}
}

func componentWithFolders(
	components []*entities.Component,
	folders []*entities.Folder,
) ([]*adapters.ComponentWithFolder, error) {
	folderIdentities := make(map[uuid.UUID]string, len(folders))
	for _, folder := range folders {
		folderIdentities[folder.ID] = folder.Identity
	}

	result := make([]*adapters.ComponentWithFolder, 0, len(components))
	for _, component := range components {
		folderIdentity, ok := folderIdentities[component.FolderID]
		if !ok {
			return nil, apperrors.Internal(
				"component_folder_not_found",
				"component references an unavailable folder",
			)
		}
		result = append(result, componentWithFolder(component, folderIdentity))
	}

	return result, nil
}

func firstComponentWithUnavailableFolder(components []*entities.Component, folders []*entities.Folder) *entities.Component {
	folderIDs := make(map[uuid.UUID]struct{}, len(folders))
	for _, folder := range folders {
		folderIDs[folder.ID] = struct{}{}
	}

	for _, component := range components {
		if _, ok := folderIDs[component.FolderID]; !ok {
			return component
		}
	}

	return nil
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
