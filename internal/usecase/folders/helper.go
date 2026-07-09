package folders

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"github.com/google/uuid"
)

func (s *Folder) resolveFolder(
	ctx context.Context,
	input *adapters.FolderIdentityInput,
	includeDeleted bool,
) (*entities.Folder, error) {
	if err := normalizeAndValidateIdentityInput(
		&input.ProjectIdentity,
		&input.EntityType,
		&input.Identity,
	); err != nil {
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}

	if includeDeleted {
		return s.folderRepository.GetByIdentityIncludingDeleted(
			ctx,
			&project.ID,
			input.EntityType,
			input.Identity,
		)
	}

	return s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
}

func (s *Folder) resolveParentID(
	ctx context.Context,
	projectID uuid.UUID,
	entityType entities.FolderEntityType,
	parentIdentity *string,
) (*uuid.UUID, error) {
	if parentIdentity == nil {
		return nil, nil
	}

	parent, err := s.folderRepository.GetByIdentity(ctx, &projectID, entityType, *parentIdentity)
	if err != nil {
		return nil, err
	}

	return uuidPointer(parent.ID), nil
}

func (s *Folder) validateNoCycle(ctx context.Context, folderID uuid.UUID, parentID *uuid.UUID) error {
	visited := make(map[uuid.UUID]struct{})
	currentID := parentID

	for currentID != nil {
		if *currentID == folderID {
			return apperrors.Conflict("folder_cycle", "folder cycle is not allowed")
		}
		if _, exists := visited[*currentID]; exists {
			return apperrors.Conflict("folder_cycle", "folder cycle is not allowed")
		}
		visited[*currentID] = struct{}{}

		parent, err := s.folderRepository.GetByIDIncludingDeleted(ctx, *currentID)
		if err != nil {
			return err
		}
		currentID = parent.ParentID
	}

	return nil
}

func normalizeAndValidateCreateInput(input *adapters.CreateFolderInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.ParentIdentity != nil {
		parentIdentity := strings.TrimSpace(*input.ParentIdentity)
		if parentIdentity == "" {
			return apperrors.InvalidInput("validation_error", "parent identity cannot be empty")
		}
		input.ParentIdentity = &parentIdentity
	}
	if err := validateFolderFields(input.ProjectIdentity, input.EntityType, input.Identity, input.DisplayName); err != nil {
		return err
	}
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	return nil
}

func normalizeAndValidateUpdateInput(input *adapters.UpdateFolderInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.ParentIdentity != nil {
		parentIdentity := strings.TrimSpace(*input.ParentIdentity)
		if parentIdentity == "" {
			return apperrors.InvalidInput("validation_error", "parent identity cannot be empty")
		}
		input.ParentIdentity = &parentIdentity
	}
	if err := validateFolderFields(input.ProjectIdentity, input.EntityType, input.Identity, input.DisplayName); err != nil {
		return err
	}
	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	return nil
}

func normalizeAndValidateListInput(input *adapters.ListFoldersInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	if input.ProjectIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}
	if !isSupportedEntityType(input.EntityType) {
		return apperrors.InvalidInput("validation_error", "unsupported folder entity type")
	}

	return nil
}

func normalizeAndValidateIdentityInput(
	projectIdentity *string,
	entityType *entities.FolderEntityType,
	identity *string,
) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*identity = strings.TrimSpace(*identity)

	if *projectIdentity == "" || *identity == "" {
		return apperrors.InvalidInput("validation_error", "project and folder identities are required")
	}
	if !isSupportedEntityType(*entityType) {
		return apperrors.InvalidInput("validation_error", "unsupported folder entity type")
	}

	return nil
}

func validateFolderFields(
	projectIdentity string,
	entityType entities.FolderEntityType,
	identity string,
	displayName string,
) error {
	if projectIdentity == "" || identity == "" || displayName == "" {
		return apperrors.InvalidInput(
			"validation_error",
			"project identity, folder identity and display name are required",
		)
	}
	if !isSupportedEntityType(entityType) {
		return apperrors.InvalidInput("validation_error", "unsupported folder entity type")
	}

	return nil
}

func isSupportedEntityType(entityType entities.FolderEntityType) bool {
	switch entityType {
	case entities.FolderEntityTypeComponents,
		entities.FolderEntityTypeConverters,
		entities.FolderEntityTypeQueries,
		entities.FolderEntityTypeDataViews:
		return true
	default:
		return false
	}
}

func uuidPointer(id uuid.UUID) *uuid.UUID {
	return &id
}

func sameNullableUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}
