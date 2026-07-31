package converters

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	relationresolver "github.com/endge-lab/service-backend/internal/usecase/relations"
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

func (c *Converter) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}

	folder, err := c.resolveFolder(ctx, projectID, *identity)
	if err != nil {
		return nil, err
	}

	return &folder.ID, nil
}

func (c *Converter) resolveFolder(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RFolder, error) {
	resolver := c.relations
	if resolver == nil {
		resolver = relationresolver.NewResolver(nil, c.folderRepository)
	}
	return resolver.ResolveFolderFromContext(ctx, identity, entities.FolderEntityTypeConverters, &projectID)
}

func converterWithFolder(converter *entities.RConverter, folderIdentity string) *ConverterWithFolder {
	return &ConverterWithFolder{
		Converter:      converter,
		FolderIdentity: folderIdentity,
	}
}

func converterWithFolders(
	converters []*entities.RConverter,
	folders []*entities.RFolder,
) ([]*ConverterWithFolder, error) {
	folderIdentities := make(map[uuid.UUID]string, len(folders))
	for _, folder := range folders {
		folderIdentities[folder.ID] = folder.Identity
	}

	result := make([]*ConverterWithFolder, 0, len(converters))
	for _, converter := range converters {
		folderIdentity, ok := folderIdentities[converter.FolderID]
		if !ok {
			return nil, apperrors.Internal(
				"converter_folder_not_found",
				"converter references an unavailable folder",
			)
		}
		result = append(result, converterWithFolder(converter, folderIdentity))
	}

	return result, nil
}

func firstConverterWithUnavailableFolder(converters []*entities.RConverter, folders []*entities.RFolder) *entities.RConverter {
	folderIDs := make(map[uuid.UUID]struct{}, len(folders))
	for _, folder := range folders {
		folderIDs[folder.ID] = struct{}{}
	}

	for _, converter := range converters {
		if _, ok := folderIDs[converter.FolderID]; !ok {
			return converter
		}
	}

	return nil
}

func (c *Converter) resolveConverter(ctx context.Context, input ConverterIdentityInput, includeDeleted bool) (*entities.RConverter, error) {
	if err := normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.ConverterIdentity); err != nil {
		return nil, err
	}
	project, err := c.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	if includeDeleted {
		return c.converterRepository.GetByIdentityIncludingDeleted(ctx, project.ID, input.ConverterIdentity)
	}
	return c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
}
func normalizeAndValidateCreateInput(input *CreateConverterInput) error {
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
func normalizeAndValidateUpdateInput(input *UpdateConverterInput) error {
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
