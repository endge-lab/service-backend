package data_views

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

func normalizeCreateInput(input *adapters.CreateDataViewInput) error {
	normalizeDataViewFields(&input.ProjectIdentity, &input.FolderIdentity, &input.QueryIdentity, &input.Identity, &input.DisplayName, &input.ViewType, &input.Description)
	normalizeDataViewPayload(&input.Source, &input.InputSchema, &input.OutputSchema, &input.Meta)
	return validateDataView(input.ProjectIdentity, input.FolderIdentity, input.QueryIdentity, input.Identity, input.DisplayName, input.ViewType)
}

func normalizeUpdateInput(input *adapters.UpdateDataViewInput) error {
	normalizeDataViewFields(&input.ProjectIdentity, &input.FolderIdentity, &input.QueryIdentity, &input.DataViewIdentity, &input.DisplayName, &input.ViewType, &input.Description)
	normalizeDataViewPayload(&input.Source, &input.InputSchema, &input.OutputSchema, &input.Meta)
	return validateDataView(input.ProjectIdentity, input.FolderIdentity, input.QueryIdentity, input.DataViewIdentity, input.DisplayName, input.ViewType)
}

func normalizeListInput(input *adapters.ListDataViewsInput) error {
	input.ProjectIdentity = strings.TrimSpace(input.ProjectIdentity)
	if input.FolderIdentity != nil {
		value := strings.TrimSpace(*input.FolderIdentity)
		input.FolderIdentity = &value
	}
	if input.QueryIdentity != nil {
		value := strings.TrimSpace(*input.QueryIdentity)
		input.QueryIdentity = &value
	}
	if input.ProjectIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}
	if input.FolderIdentity != nil && *input.FolderIdentity == "" {
		return apperrors.InvalidInput("validation_error", "folder identity cannot be empty")
	}
	if input.QueryIdentity != nil && *input.QueryIdentity == "" {
		return apperrors.InvalidInput("validation_error", "query identity cannot be empty")
	}
	return nil
}

func normalizeIdentities(projectIdentity, dataViewIdentity *string) error {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*dataViewIdentity = strings.TrimSpace(*dataViewIdentity)
	if *projectIdentity == "" || *dataViewIdentity == "" {
		return apperrors.InvalidInput("validation_error", "project and data view identities are required")
	}
	return nil
}

func normalizeDataViewFields(projectIdentity, folderIdentity, queryIdentity, identity, displayName, viewType *string, description **string) {
	*projectIdentity = strings.TrimSpace(*projectIdentity)
	*folderIdentity = strings.TrimSpace(*folderIdentity)
	*queryIdentity = strings.TrimSpace(*queryIdentity)
	*identity = strings.TrimSpace(*identity)
	*displayName = strings.TrimSpace(*displayName)
	*viewType = strings.TrimSpace(*viewType)
	if *description != nil {
		value := strings.TrimSpace(**description)
		*description = &value
	}
}

func normalizeDataViewPayload(source, inputSchema, outputSchema, meta *map[string]any) {
	if *source == nil {
		*source = map[string]any{}
	}
	if *inputSchema == nil {
		*inputSchema = map[string]any{}
	}
	if *outputSchema == nil {
		*outputSchema = map[string]any{}
	}
	if *meta == nil {
		*meta = map[string]any{}
	}
}

func validateDataView(projectIdentity, folderIdentity, queryIdentity, identity, displayName, viewType string) error {
	if projectIdentity == "" || folderIdentity == "" || queryIdentity == "" || identity == "" || displayName == "" || viewType == "" {
		return apperrors.InvalidInput("validation_error", "project, folder, query, identity, display name and view type are required")
	}
	return nil
}

func (s *DataView) resolveFolderID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}
	folder, err := s.resolveFolder(ctx, projectID, *identity)
	if err != nil {
		return nil, err
	}
	return &folder.ID, nil
}

func (s *DataView) resolveFolder(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Folder, error) {
	folder, err := s.folderRepository.GetByIdentity(ctx, &projectID, entities.FolderEntityTypeDataViews, identity)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have data_views entity type")
	}
	return folder, err
}

func (s *DataView) resolveQueryID(ctx context.Context, projectID uuid.UUID, identity *string) (*uuid.UUID, error) {
	if identity == nil {
		return nil, nil
	}
	query, err := s.resolveActiveQuery(ctx, projectID, *identity)
	if err != nil {
		return nil, err
	}
	return &query.ID, nil
}

func (s *DataView) resolveActiveQuery(ctx context.Context, projectID uuid.UUID, identity string) (*entities.Query, error) {
	query, err := s.queryRepository.GetByIdentity(ctx, projectID, identity)
	if !errors.Is(err, apperrors.ErrNotFound) {
		return query, err
	}
	existsOutside, existsErr := s.queryRepository.ExistsActiveByIdentityOutsideProject(ctx, projectID, identity)
	if existsErr != nil {
		return nil, existsErr
	}
	if existsOutside {
		return nil, apperrors.InvalidInput("query_project_mismatch", "query must belong to the project")
	}
	return nil, err
}

func dataViewFromCreate(projectID, folderID, queryID uuid.UUID, input adapters.CreateDataViewInput) *entities.DataView {
	return &entities.DataView{ProjectID: projectID, FolderID: folderID, QueryID: queryID, Identity: input.Identity, DisplayName: input.DisplayName, Description: input.Description, ViewType: input.ViewType, Source: input.Source, InputSchema: input.InputSchema, OutputSchema: input.OutputSchema, Meta: input.Meta, Active: input.Active}
}

func dataViewFromUpdate(current *entities.DataView, folderID, queryID uuid.UUID, input adapters.UpdateDataViewInput) *entities.DataView {
	return &entities.DataView{ID: current.ID, ProjectID: current.ProjectID, FolderID: folderID, QueryID: queryID, Identity: current.Identity, DisplayName: input.DisplayName, Description: input.Description, ViewType: input.ViewType, Source: input.Source, InputSchema: input.InputSchema, OutputSchema: input.OutputSchema, Meta: input.Meta, Active: input.Active, DeletedAt: current.DeletedAt, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt}
}

func dataViewWithRelations(dataView *entities.DataView, folderIdentity, queryIdentity string) *adapters.DataViewWithRelations {
	return &adapters.DataViewWithRelations{DataView: dataView, FolderIdentity: folderIdentity, QueryIdentity: queryIdentity}
}

func dataViewsWithRelations(dataViews []*entities.DataView, folders []*entities.Folder, queries []*entities.Query) ([]*adapters.DataViewWithRelations, error) {
	folderIdentities := make(map[uuid.UUID]string, len(folders))
	for _, folder := range folders {
		folderIdentities[folder.ID] = folder.Identity
	}
	queryIdentities := make(map[uuid.UUID]string, len(queries))
	for _, query := range queries {
		queryIdentities[query.ID] = query.Identity
	}
	result := make([]*adapters.DataViewWithRelations, 0, len(dataViews))
	for _, dataView := range dataViews {
		folderIdentity, folderOK := folderIdentities[dataView.FolderID]
		if !folderOK {
			return nil, apperrors.Internal("data_view_folder_not_found", "data view references an unavailable folder")
		}
		queryIdentity, queryOK := queryIdentities[dataView.QueryID]
		if !queryOK {
			return nil, apperrors.Internal("data_view_query_not_found", "data view references an unavailable query")
		}
		result = append(result, dataViewWithRelations(dataView, folderIdentity, queryIdentity))
	}
	return result, nil
}
