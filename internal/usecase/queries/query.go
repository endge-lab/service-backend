package queries

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const queryOperationTimeout = 15 * time.Second

type Query struct {
	queryRepository   ports.QueriesRepository
	folderRepository  ports.FoldersRepository
	projectRepository ports.ProjectsRepository
	observer          observability.Observer
}

type QueryParams struct {
	QueryRepository   ports.QueriesRepository
	FolderRepository  ports.FoldersRepository
	ProjectRepository ports.ProjectsRepository
	Observability     *observability.Core
	Metrics           *shared.UseCaseMetrics
}

func NewQueryService(params QueryParams) *Query {
	return &Query{queryRepository: params.QueryRepository, folderRepository: params.FolderRepository, projectRepository: params.ProjectRepository, observer: params.Observability.For(observability.LayerUseCase, "queries_usecase").WithRecorder(params.Metrics)}
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	folder, err := s.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
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
	query, err := s.queryRepository.Create(ctx, queryFromCreate(project.WorkspaceID, project.ID, folder.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.Identity))
		return nil, err
	}
	observed.Logger().Debug("query created", zap.String("query_id", query.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", query.Identity))
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	current, err := s.queryRepository.GetByIdentity(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	folder, err := s.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
	query, err := s.queryRepository.Update(ctx, queryFromUpdate(current, folder.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	observed.Logger().Debug("query updated", zap.String("query_id", query.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", query.Identity))
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	query, err := s.queryRepository.GetByIdentity(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	folder, err := s.folderRepository.GetByID(ctx, query.FolderID)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_id", query.ID.String()))
		return nil, err
	}
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	folderID, err := s.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	values, err := s.queryRepository.List(ctx, ports.QueriesFilter{ProjectID: project.ID, FolderID: folderID, QueryType: input.QueryType})
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	if len(values) == 0 {
		return []*QueryWithFolder{}, nil
	}
	folders, err := s.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeQueries)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	return queriesWithFolders(values, folders)
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	folderID, err := s.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	return s.queryRepository.Count(ctx, ports.QueriesFilter{ProjectID: project.ID, FolderID: folderID, QueryType: input.QueryType})
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
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return err
	}
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
	if err = change(ctx, query.ID); err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_id", query.ID.String()))
		return err
	}
	observed.Logger().Debug("query state changed", zap.String("query_id", query.ID.String()), zap.String("operation", op))
	return nil
}
