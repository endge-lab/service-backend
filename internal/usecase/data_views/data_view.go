package data_views

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

const dataViewOperationTimeout = 15 * time.Second

type DataView struct {
	dataViewRepository ports.DataViewsRepository
	queryRepository    ports.QueriesRepository
	folderRepository   ports.FoldersRepository
	projectRepository  ports.ProjectsRepository
	observer           observability.Observer
}

type DataViewParams struct {
	DataViewRepository ports.DataViewsRepository
	QueryRepository    ports.QueriesRepository
	FolderRepository   ports.FoldersRepository
	ProjectRepository  ports.ProjectsRepository
	Observability      *observability.Core
	Metrics            *shared.UseCaseMetrics
}

func NewDataViewService(params DataViewParams) *DataView {
	return &DataView{dataViewRepository: params.DataViewRepository, queryRepository: params.QueryRepository, folderRepository: params.FolderRepository, projectRepository: params.ProjectRepository, observer: params.Observability.For(observability.LayerUseCase, "data_views_usecase").WithRecorder(params.Metrics)}
}

// Create создает DataView в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - данные DataView, identity проекта, папки и Query
//
// Что делает функция:
//
//	Валидирует input, разрешает проект, папку data_views и активную Query проекта,
//	проверяет уникальность identity и сохраняет DataView в репозитории.
//
// Возвращаемые значения:
//
//	*DataViewWithRelations - созданный DataView и identity связанных сущностей
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *DataView) Create(ctx context.Context, input CreateDataViewInput) (result *DataViewWithRelations, err error) {
	const op = "data_view.create"
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
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
	query, err := s.resolveActiveQuery(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	exists, err := s.dataViewRepository.ExistsByIdentity(ctx, project.ID, input.Identity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("data_view_identity", input.Identity))
		return nil, err
	}
	if exists {
		err = apperrors.Conflict("identity_conflict", "data view identity already exists")
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("data_view_identity", input.Identity))
		return nil, err
	}
	dataView, err := s.dataViewRepository.Create(ctx, dataViewFromCreate(project.WorkspaceID, project.ID, folder.ID, query.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity), zap.String("data_view_identity", input.Identity))
		return nil, err
	}
	observed.Logger().Debug("data view created", zap.String("data_view_id", dataView.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("data_view_identity", dataView.Identity), zap.String("folder_identity", folder.Identity), zap.String("query_identity", query.Identity))
	return dataViewWithRelations(dataView, folder.Identity, query.Identity), nil
}

// Update обновляет editable поля DataView и его связи в пределах проекта.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - identity проекта, DataView, папки, Query и editable поля
//
// Что делает функция:
//
//	Валидирует input, разрешает проект, существующий DataView, папку data_views и
//	активную Query проекта, затем сохраняет изменения с неизменяемым identity.
//
// Возвращаемые значения:
//
//	*DataViewWithRelations - обновленный DataView и identity связанных сущностей
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *DataView) Update(ctx context.Context, input UpdateDataViewInput) (result *DataViewWithRelations, err error) {
	const op = "data_view.update"
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
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
	current, err := s.dataViewRepository.GetByIdentity(ctx, project.ID, input.DataViewIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("data_view_identity", input.DataViewIdentity))
		return nil, err
	}
	folder, err := s.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
	query, err := s.resolveActiveQuery(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_identity", input.QueryIdentity))
		return nil, err
	}
	dataView, err := s.dataViewRepository.Update(ctx, dataViewFromUpdate(current, folder.ID, query.ID, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("data_view_identity", input.DataViewIdentity))
		return nil, err
	}
	observed.Logger().Debug("data view updated", zap.String("data_view_id", dataView.ID.String()), zap.String("project_identity", input.ProjectIdentity), zap.String("data_view_identity", dataView.Identity), zap.String("folder_identity", folder.Identity), zap.String("query_identity", query.Identity))
	return dataViewWithRelations(dataView, folder.Identity, query.Identity), nil
}

// GetByIdentity возвращает активный DataView и identity его связей.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - identity проекта и DataView
//
// Что делает функция:
//
//	Валидирует identities, получает проект и DataView, затем отдельно разрешает его
//	папку и Query, включая soft-deleted Query.
//
// Возвращаемые значения:
//
//	*DataViewWithRelations - активный DataView и identity связанных сущностей
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *DataView) GetByIdentity(ctx context.Context, input GetDataViewInput) (result *DataViewWithRelations, err error) {
	const op = "data_view.get_by_identity"
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeIdentities(&input.ProjectIdentity, &input.DataViewIdentity); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	dataView, err := s.dataViewRepository.GetByIdentity(ctx, project.ID, input.DataViewIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("data_view_identity", input.DataViewIdentity))
		return nil, err
	}
	folder, err := s.folderRepository.GetByID(ctx, dataView.FolderID)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("folder_id", dataView.FolderID.String()))
		return nil, err
	}
	query, err := s.queryRepository.GetByIDIncludingDeleted(ctx, dataView.QueryID)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("query_id", dataView.QueryID.String()))
		return nil, err
	}
	return dataViewWithRelations(dataView, folder.Identity, query.Identity), nil
}

// List возвращает активные DataView проекта по optional фильтрам.
//
// Параметры:
//
//	ctx - контекст выполнения операции
//	input - identity проекта и optional identity папки и Query
//
// Что делает функция:
//
//	Валидирует input, разрешает фильтры, получает список DataView, папок и активных
//	Query одним запросом на сущность, затем соединяет связи в памяти без N+1 запросов.
//
// Возвращаемые значения:
//
//	[]*DataViewWithRelations - список DataView и identity связанных сущностей
//	error - ошибка валидации, разрешения зависимостей или хранения
func (s *DataView) List(ctx context.Context, input ListDataViewsInput) (result []*DataViewWithRelations, err error) {
	const op = "data_view.list"
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
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
	queryID, err := s.resolveQueryID(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	values, err := s.dataViewRepository.List(ctx, ports.DataViewsFilter{ProjectID: project.ID, FolderID: folderID, QueryID: queryID})
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	if len(values) == 0 {
		return []*DataViewWithRelations{}, nil
	}
	folders, err := s.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeDataViews)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	queries, err := s.queryRepository.List(ctx, ports.QueriesFilter{ProjectID: project.ID})
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	return dataViewsWithRelations(values, folders, queries)
}

// SoftDelete помечает активный DataView удаленным.
func (s *DataView) SoftDelete(ctx context.Context, input DataViewIdentityInput) error {
	return s.change(ctx, "data_view.soft_delete", input, false, s.dataViewRepository.SoftDelete)
}

// Restore восстанавливает soft-deleted DataView.
func (s *DataView) Restore(ctx context.Context, input DataViewIdentityInput) error {
	return s.change(ctx, "data_view.restore", input, true, s.dataViewRepository.Restore)
}

// HardDelete физически удаляет soft-deleted DataView.
func (s *DataView) HardDelete(ctx context.Context, input DataViewIdentityInput) error {
	return s.change(ctx, "data_view.hard_delete", input, true, s.dataViewRepository.HardDelete)
}

// Count возвращает количество активных DataView по optional фильтрам.
func (s *DataView) Count(ctx context.Context, input ListDataViewsInput) (count int64, err error) {
	const op = "data_view.count"
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
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
	queryID, err := s.resolveQueryID(ctx, project.ID, input.QueryIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return 0, err
	}
	return s.dataViewRepository.Count(ctx, ports.DataViewsFilter{ProjectID: project.ID, FolderID: folderID, QueryID: queryID})
}

func (s *DataView) change(ctx context.Context, op string, input DataViewIdentityInput, includeDeleted bool, change func(context.Context, uuid.UUID) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, dataViewOperationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeIdentities(&input.ProjectIdentity, &input.DataViewIdentity); err != nil {
		logOperationError(observed.Logger(), op, err)
		return err
	}
	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return err
	}
	var dataView *entities.RDataView
	if includeDeleted {
		dataView, err = s.dataViewRepository.GetByIdentityIncludingDeleted(ctx, project.ID, input.DataViewIdentity)
	} else {
		dataView, err = s.dataViewRepository.GetByIdentity(ctx, project.ID, input.DataViewIdentity)
	}
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("data_view_identity", input.DataViewIdentity))
		return err
	}
	if err = change(ctx, dataView.ID); err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("data_view_id", dataView.ID.String()))
		return err
	}
	observed.Logger().Debug("data view state changed", zap.String("data_view_id", dataView.ID.String()), zap.String("operation", op))
	return nil
}
