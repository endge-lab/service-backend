package queries

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	relationresolver "github.com/endge-lab/service-backend/internal/usecase/relations"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const queryOperationTimeout = 15 * time.Second

type Query struct {
	queryRepository  ports.QueriesRepository
	folderRepository ports.FoldersRepository
	relations        *relationresolver.Resolver
	observer         observability.Observer
}

type QueryParams struct {
	QueryRepository   ports.QueriesRepository
	FolderRepository  ports.FoldersRepository
	ProjectRepository ports.ProjectsRepository
	Relations         *relationresolver.Resolver
	Observability     *observability.Core
	Metrics           *shared.UseCaseMetrics
}

func NewQueryService(params QueryParams) *Query {
	resolver := params.Relations
	if resolver == nil {
		resolver = relationresolver.NewResolver(params.ProjectRepository, params.FolderRepository)
	}
	return &Query{queryRepository: params.QueryRepository, folderRepository: params.FolderRepository, relations: resolver, observer: params.Observability.For(observability.LayerUseCase, "queries_usecase").WithRecorder(params.Metrics)}
}

// Create создает Query в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - данные Query, identity проекта и папки
//
// Что делает функция:
//
//	Валидирует и нормализует input, разрешает проект и папку с entity type queries,
//	проверяет уникальность identity внутри проекта и сохраняет Query в репозитории.
//
// Возвращаемые значения:
//
//	*QueryWithFolder - созданная Query и identity ее папки
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *Query) Create(ctx context.Context, input CreateQueryInput) (result *QueryWithFolder, err error) {
	const op = "query.create"

	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeCreateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "query create input validated", nil, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.Identity))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query create", nil, zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	folder, err := s.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
	observed.RecordStep(op+".folder_resolved", "folder resolved for query create", nil, zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))
	exists, err := s.queryRepository.ExistsByIdentity(ctx, project.ID, input.Identity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.Identity))
		return nil, err
	}
	if exists {
		err = apperrors.Conflict("identity_conflict", "query identity already exists")
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.Identity))
		return nil, err
	}
	observed.RecordStep(op+".identity_available", "query identity availability confirmed", nil, zap.String("query_identity", input.Identity))
	query, err := s.queryRepository.Create(ctx, queryFromCreate(project.WorkspaceID, project.ID, folder.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.Identity))
		return nil, err
	}
	observed.RecordStep(op+".persisted", "query created", nil, zap.String("query_id", query.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", query.Identity))
	return queryWithFolder(query, folder.Identity), nil
}

// Update обновляет editable поля Query и ее папку в пределах проекта.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - identity проекта, Query, новой папки и editable поля
//
// Что делает функция:
//
//	Валидирует input, разрешает проект, существующую Query и папку queries,
//	сохраняет обновленную Query с неизменяемым identity.
//
// Возвращаемые значения:
//
//	*QueryWithFolder - обновленная Query и identity ее папки
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *Query) Update(ctx context.Context, input UpdateQueryInput) (result *QueryWithFolder, err error) {
	const op = "query.update"
	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeUpdateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "query update input validated", nil, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query update", nil, zap.String("project_id", project.ID.String()))
	current, err := s.queryRepository.GetByIdentity(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	observed.RecordStep(op+".current_resolved", "query resolved for update", nil, zap.String("query_id", current.ID.String()), zap.String("query_identity", current.Identity))
	folder, err := s.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
	observed.RecordStep(op+".folder_resolved", "folder resolved for query update", nil, zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))
	query, err := s.queryRepository.Update(ctx, queryFromUpdate(current, folder.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	observed.RecordStep(op+".persisted", "query updated", nil, zap.String("query_id", query.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", query.Identity))
	return queryWithFolder(query, folder.Identity), nil
}

// GetByIdentity возвращает активную Query по identity в пределах проекта.
func (s *Query) GetByIdentity(ctx context.Context, input GetQueryInput) (result *QueryWithFolder, err error) {
	const op = "query.get_by_identity"
	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeIdentities(&input.ProjectIdentity, &input.QueryIdentity); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "query identity input validated", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query retrieval", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	query, err := s.queryRepository.GetByIdentity(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	observed.RecordStep(op+".current_resolved", "query resolved for retrieval", nil,
		zap.String("query_id", query.ID.String()), zap.String("query_identity", query.Identity))
	folder, err := s.folderRepository.GetByID(ctx, query.FolderID)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_id", query.ID.String()))
		return nil, err
	}
	observed.RecordStep(op+".folder_resolved", "query folder resolved for retrieval", nil,
		zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))
	observed.RecordStep(op+".result_loaded", "query retrieved", nil, zap.String("query_id", query.ID.String()), zap.String("query_identity", query.Identity), zap.String("folder_id", folder.ID.String()))
	return queryWithFolder(query, folder.Identity), nil
}

// List возвращает активные Query проекта по optional фильтрам.
func (s *Query) List(ctx context.Context, input ListQueriesInput) (result []*QueryWithFolder, err error) {
	const op = "query.list"
	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeListInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "query list input validated", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", dereferenceString(input.FolderIdentity)))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query list", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	folderID, err := s.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".folder_filter_resolved", "query folder filter resolved", nil,
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)))
	values, err := s.queryRepository.List(ctx, ports.QueriesFilter{ProjectID: project.ID, FolderID: folderID, QueryType: input.QueryType})
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	if len(values) == 0 {
		observed.RecordStep(op+".result_loaded", "queries listed", nil, zap.String("project_identity", input.ProjectIdentity), zap.Int("count", 0))
		return []*QueryWithFolder{}, nil
	}
	folders, err := s.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeQueries)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	result, err = queriesWithFolders(values, folders)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(op+".result_loaded", "queries listed", nil, zap.String("project_identity", input.ProjectIdentity), zap.Int("count", len(result)))
	return result, nil
}

// SoftDelete помечает активную Query удаленной.
func (s *Query) SoftDelete(ctx context.Context, input QueryIdentityInput) (err error) {
	return s.change(ctx, "query.soft_delete", input, false, s.queryRepository.SoftDelete)
}

// Restore восстанавливает soft-deleted Query.
func (s *Query) Restore(ctx context.Context, input QueryIdentityInput) (err error) {
	return s.change(ctx, "query.restore", input, true, s.queryRepository.Restore)
}

// HardDelete физически удаляет soft-deleted Query вместе со связанными DataView.
func (s *Query) HardDelete(ctx context.Context, input QueryIdentityInput) (err error) {
	return s.change(ctx, "query.hard_delete", input, true, s.queryRepository.HardDelete)
}

// Count возвращает количество активных Query по optional фильтрам.
func (s *Query) Count(ctx context.Context, input ListQueriesInput) (count int64, err error) {
	const op = "query.count"
	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeListInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return 0, err
	}
	observed.RecordStep(op+".input_validated", "query count input validated", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", dereferenceString(input.FolderIdentity)))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query count", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	folderID, err := s.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	observed.RecordStep(op+".folder_filter_resolved", "query folder filter resolved", nil,
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)))
	count, err = s.queryRepository.Count(ctx, ports.QueriesFilter{ProjectID: project.ID, FolderID: folderID, QueryType: input.QueryType})
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	observed.RecordStep(op+".result_loaded", "queries counted", nil, zap.String("project_identity", input.ProjectIdentity), zap.Int64("count", count))
	return count, nil
}

func (s *Query) change(ctx context.Context, op string, input QueryIdentityInput, includeDeleted bool, change func(context.Context, uuid.UUID) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, queryOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeIdentities(&input.ProjectIdentity, &input.QueryIdentity); err != nil {
		logOperationError(observed.Logger(), op, err)
		return err
	}
	observed.RecordStep(op+".input_validated", "query state change input validated", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
	project, err := s.relations.ResolveProjectFromContext(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for query state change", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	var query *entities.RQuery
	if includeDeleted {
		query, err = s.queryRepository.GetByIdentityIncludingDeleted(ctx, project.ID, input.QueryIdentity)
	} else {
		query, err = s.queryRepository.GetByIdentity(ctx, project.ID, input.QueryIdentity)
	}
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return err
	}
	observed.RecordStep(op+".current_resolved", "query resolved for state change", nil,
		zap.String("query_id", query.ID.String()), zap.String("query_identity", query.Identity))
	if err = change(ctx, query.ID); err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_id", query.ID.String()))
		return err
	}
	observed.RecordStep(op+".persisted", "query state changed", nil, zap.String("query_id", query.ID.String()), zap.String("operation", op))
	return nil
}
