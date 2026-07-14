package converters

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
)

func (c *Converter) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}
	folder, err := c.folderRepository.GetByIdentity(ctx, &projectID, entities.FolderEntityTypeConverters, *identity)
	if err != nil {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have converters entity type")
	}
	return &folder.ID, nil
}

func (c *Converter) resolveConverter(ctx context.Context, input adapters.ConverterIdentityInput, includeDeleted bool) (*entities.Converter, error) {
	if err := normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.ConverterIdentity); err != nil {
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	if includeDeleted {
		return c.converterRepository.GetByIdentityIncludingDeleted(ctx, project.ID, input.ConverterIdentity)
	}
	return c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
}
func normalizeAndValidateCreateInput(input *adapters.CreateConverterInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.FolderIdentity = strings.TrimSpace(input.FolderIdentity)
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ConverterType = strings.TrimSpace(input.ConverterType)
	if input.Description != nil {
		v := strings.TrimSpace(*input.Description)
		input.Description = &v
	}
	if err := validateFields(input.ProjectIdentity, input.FolderIdentity, input.Identity, input.DisplayName, input.ConverterType); err != nil {
		return err
	}
	if input.Source == nil {
		input.Source = map[string]any{}
	}
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}
	return nil
}
func normalizeAndValidateUpdateInput(input *adapters.UpdateConverterInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.ConverterIdentity = strings.TrimSpace(input.ConverterIdentity)
	input.FolderIdentity = strings.TrimSpace(input.FolderIdentity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ConverterType = strings.TrimSpace(input.ConverterType)
	if input.Description != nil {
		v := strings.TrimSpace(*input.Description)
		input.Description = &v
	}
	if err := validateFields(input.ProjectIdentity, input.FolderIdentity, input.ConverterIdentity, input.DisplayName, input.ConverterType); err != nil {
		return err
	}
	if input.Source == nil {
		input.Source = map[string]any{}
	}
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}
	return nil
}
func normalizeAndValidateIdentityInput(projectIdentity, converterIdentity *string) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*converterIdentity = strings.TrimSpace(*converterIdentity)
	if *projectIdentity == "" || *converterIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project identity and converter identity are required")
	}
	return nil
}
func normalizeAndValidateListInput(projectIdentity *string, folderIdentity *string) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	if *projectIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}
	if folderIdentity != nil {
		v := strings.TrimSpace(*folderIdentity)
		*folderIdentity = v
	}
	return nil
}
func validateFields(project, folder, identity, displayName, converterType string) error {
	if project == "" || folder == "" || identity == "" || displayName == "" || converterType == "" {
		return apperrors.InvalidInput("validation_error", "project identity, folder identity, converter identity, display name and converter type are required")
	}
	return nil
}
