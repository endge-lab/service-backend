package queries

import (
	"context"
	"errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
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

func normalizeCreateInput(input *CreateQueryInput) error {
	normalizeQueryFields(&input.ProjectIdentity, &input.FolderIdentity, &input.Identity, &input.DisplayName, &input.QueryType, &input.Description)
	normalizeQueryPayload(&input.Source, &input.Params, &input.Headers, &input.Meta)
	return validateQuery(input.ProjectIdentity, input.FolderIdentity, input.Identity, input.DisplayName, input.QueryType, input.TimeoutMS)
}

func normalizeUpdateInput(input *UpdateQueryInput) error {
	normalizeQueryFields(&input.ProjectIdentity, &input.FolderIdentity, &input.QueryIdentity, &input.DisplayName, &input.QueryType, &input.Description)
	normalizeQueryPayload(&input.Source, &input.Params, &input.Headers, &input.Meta)
	return validateQuery(input.ProjectIdentity, input.FolderIdentity, input.QueryIdentity, input.DisplayName, input.QueryType, input.TimeoutMS)
}

func normalizeListInput(input *ListQueriesInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	if input.FolderIdentity != nil {
		value := strings.TrimSpace(*input.FolderIdentity)
		input.FolderIdentity = &value
	}
	if input.QueryType != nil {
		value := strings.TrimSpace(*input.QueryType)
		input.QueryType = &value
	}
	if input.ProjectIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}
	if input.FolderIdentity != nil && *input.FolderIdentity == "" {
		return apperrors.InvalidInput("validation_error", "folder identity cannot be empty")
	}
	if input.QueryType != nil && *input.QueryType == "" {
		return apperrors.InvalidInput("validation_error", "query type cannot be empty")
	}
	return nil
}

func normalizeIdentities(projectIdentity, queryIdentity *string) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*queryIdentity = strings.TrimSpace(*queryIdentity)
	if *projectIdentity == "" || *queryIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project and query identities are required")
	}
	return nil
}

func normalizeQueryFields(projectIdentity, folderIdentity, identity, displayName, queryType *string, description **string) {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*folderIdentity = strings.TrimSpace(*folderIdentity)
	*identity = strings.TrimSpace(*identity)
	*displayName = strings.TrimSpace(*displayName)
	*queryType = strings.TrimSpace(*queryType)
	if *description != nil {
		value := strings.TrimSpace(**description)
		*description = &value
	}
}

func normalizeQueryPayload(source *map[string]any, params *[]any, headers, meta *map[string]any) {
	if *source == nil {
		*source = map[string]any{}
	}
	if *params == nil {
		*params = []any{}
	}
	if *headers == nil {
		*headers = map[string]any{}
	}
	if *meta == nil {
		*meta = map[string]any{}
	}
}

func validateQuery(projectIdentity, folderIdentity, identity, displayName, queryType string, timeoutMS *int) error {
	if projectIdentity == "" || folderIdentity == "" || identity == "" || displayName == "" || queryType == "" {
		return apperrors.InvalidInput("validation_error", "project, folder, identity, display name and query type are required")
	}
	if timeoutMS != nil && *timeoutMS <= 0 {
		return apperrors.InvalidInput("validation_error", "timeout must be positive")
	}
	return nil
}

func (s *Query) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}
	folder, err := s.resolveFolder(ctx, projectID, *identity)
	if err != nil {
		return nil, err
	}
	return &folder.ID, nil
}

func (s *Query) resolveFolder(ctx context.Context, projectID uuid.UUID, identity string) (*entities.RFolder, error) {
	folder, err := s.folderRepository.GetByIdentity(ctx, &projectID, entities.FolderEntityTypeQueries, identity)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have queries entity type")
	}
	return folder, err
}

func queryFromCreate(projectID, folderID uuid.UUID, input CreateQueryInput) *entities.RQuery {
	return &entities.RQuery{ProjectID: projectID, FolderID: folderID, Identity: input.Identity, DisplayName: input.DisplayName, Description: input.Description, QueryType: input.QueryType, Source: input.Source, Params: input.Params, Headers: input.Headers, Auth: input.Auth, TimeoutMS: input.TimeoutMS, MockData: input.MockData, MockDataEnabled: input.MockDataEnabled, Meta: input.Meta, Active: input.Active}
}

func queryFromUpdate(current *entities.RQuery, folderID uuid.UUID, input UpdateQueryInput) *entities.RQuery {
	return &entities.RQuery{ID: current.ID, ProjectID: current.ProjectID, FolderID: folderID, Identity: current.Identity, DisplayName: input.DisplayName, Description: input.Description, QueryType: input.QueryType, Source: input.Source, Params: input.Params, Headers: input.Headers, Auth: input.Auth, TimeoutMS: input.TimeoutMS, MockData: input.MockData, MockDataEnabled: input.MockDataEnabled, Meta: input.Meta, Active: input.Active, DeletedAt: current.DeletedAt, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt}
}

func queryWithFolder(query *entities.RQuery, folderIdentity string) *QueryWithFolder {
	return &QueryWithFolder{Query: query, FolderIdentity: folderIdentity}
}

func queriesWithFolders(queries []*entities.RQuery, folders []*entities.RFolder) ([]*QueryWithFolder, error) {
	identities := make(map[uuid.UUID]string, len(folders))
	for _, folder := range folders {
		identities[folder.ID] = folder.Identity
	}
	result := make([]*QueryWithFolder, 0, len(queries))
	for _, query := range queries {
		identity, ok := identities[query.FolderID]
		if !ok {
			return nil, apperrors.Internal("query_folder_not_found", "query references an unavailable folder")
		}
		result = append(result, queryWithFolder(query, identity))
	}
	return result, nil
}
