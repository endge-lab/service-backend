package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var _ ports.DataViewsRepository = (*DataViewsRepository)(nil)

type DataViewsRepository struct{ *baseRepository }

func NewDataViewsRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *DataViewsRepository {
	return &DataViewsRepository{baseRepository: newBaseRepository(queries, tracer, logger, "data_views")}
}

// Create сохраняет новый DataView в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	dataView - DataView с уже разрешенными UUID проекта, папки и Query
//
// Что делает функция:
//
//	Преобразует domain entity в SQLC-параметры, выполняет INSERT и маппит сохраненную строку.
//
// Возвращаемые значения:
//
//	*entities.DataView - DataView с полями, заполненными хранилищем
//	error - storage error, преобразованная в domain error
func (r *DataViewsRepository) Create(ctx context.Context, dataView *entities.DataView) (result *entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.Create")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).CreateDataView(ctx, mappers.CreateDataViewParams(dataView))
	if err != nil {
		return nil, r.mapWriteError(err, "create data view failed")
	}

	return mappers.DataView(value), nil
}

// GetByID возвращает активный DataView по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID DataView
//
// Что делает функция:
//
//	Выполняет SQLC-выборку, исключающую soft-deleted запись, и маппит результат в domain entity.
//
// Возвращаемые значения:
//
//	*entities.DataView - найденный активный DataView
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.GetByID")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetDataViewByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get data view by id failed")
	}

	return mappers.DataView(value), nil
}

// GetByIdentity возвращает активный DataView по identity внутри проекта.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID проекта-владельца DataView
//	identity - человекочитаемый идентификатор DataView
//
// Что делает функция:
//
//	Выполняет project-scoped SQLC-выборку и исключает soft-deleted запись.
//
// Возвращаемые значения:
//
//	*entities.DataView - найденный активный DataView
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.GetByIdentity")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetDataViewByIdentity(ctx, sqlc.GetDataViewByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get data view by identity failed")
	}

	return mappers.DataView(value), nil
}

// GetByIdentityIncludingDeleted возвращает DataView по identity внутри проекта с учетом soft-delete.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID проекта-владельца DataView
//	identity - человекочитаемый идентификатор DataView
//
// Что делает функция:
//
//	Получает DataView без фильтра deletedAt для restore и hard-delete flow.
//
// Возвращаемые значения:
//
//	*entities.DataView - найденный DataView, включая soft-deleted запись
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.GetByIdentityIncludingDeleted")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetDataViewByIdentityIncludingDeleted(ctx, sqlc.GetDataViewByIdentityIncludingDeletedParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get data view by identity including deleted failed")
	}

	return mappers.DataView(value), nil
}

// List возвращает активные DataView проекта по optional фильтрам папки и Query.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	filter - UUID проекта и optional UUID папки и Query
//
// Что делает функция:
//
//	Передает фильтры в SQLC и исключает soft-deleted DataView и DataView со soft-deleted Query.
//
// Возвращаемые значения:
//
//	[]*entities.DataView - список активных DataView
//	error - внутренняя ошибка хранения
func (r *DataViewsRepository) List(ctx context.Context, filter ports.DataViewsFilter) (result []*entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.List")
	defer func() { step.End(err) }()

	values, err := r.queries(ctx).ListDataViews(ctx, sqlc.ListDataViewsParams{
		ProjectID: filter.ProjectID,
		FolderID:  mappers.NullableUUIDToSQLC(filter.FolderID),
		QueryID:   mappers.NullableUUIDToSQLC(filter.QueryID),
	})
	if err != nil {
		r.logger.Error("list data views failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list data views")
	}

	result = make([]*entities.DataView, 0, len(values))
	for _, value := range values {
		result = append(result, mappers.DataView(value))
	}

	return result, nil
}

// Update сохраняет editable payload активного DataView.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	dataView - DataView с UUID и обновленными editable полями
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE, который сохраняет identity и обновляет updatedAt на стороне БД.
//
// Возвращаемые значения:
//
//	*entities.DataView - обновленный активный DataView
//	error - not_found, validation error или внутренняя ошибка хранения
func (r *DataViewsRepository) Update(ctx context.Context, dataView *entities.DataView) (result *entities.DataView, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.Update")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).UpdateDataView(ctx, mappers.UpdateDataViewParams(dataView))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "data view not found")
		}
		return nil, r.mapWriteError(err, "update data view failed")
	}

	return mappers.DataView(value), nil
}

// SoftDelete помечает активный DataView удаленным по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID активного DataView
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE deletedAt и updatedAt; проверяет, что была изменена одна или более строк.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.data_views.SoftDelete", "soft delete data view failed", id, r.queries(ctx).SoftDeleteDataView)
}

// Restore восстанавливает soft-deleted DataView по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID soft-deleted DataView
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE, очищающий deletedAt и обновляющий updatedAt.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.data_views.Restore", "restore data view failed", id, r.queries(ctx).RestoreDataView)
}

// HardDelete физически удаляет DataView по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID soft-deleted DataView
//
// Что делает функция:
//
//	Выполняет DELETE независимо от soft-delete состояния записи.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *DataViewsRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.data_views.HardDelete", "hard delete data view failed", id, r.queries(ctx).HardDeleteDataView)
}

// ExistsByIdentity проверяет наличие DataView identity внутри проекта.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID проекта
//	identity - проверяемый identity
//
// Что делает функция:
//
//	Выполняет SQLC EXISTS, учитывающий active и soft-deleted записи для соблюдения unique constraint.
//
// Возвращаемые значения:
//
//	bool - признак существования DataView
//	error - внутренняя ошибка хранения
func (r *DataViewsRepository) ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.ExistsByIdentity")
	defer func() { step.End(err) }()

	exists, err = r.queries(ctx).ExistsDataViewByIdentity(ctx, sqlc.ExistsDataViewByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		r.logger.Error("exists data view by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check data view identity")
	}

	return exists, nil
}

// Count возвращает количество активных DataView проекта по optional фильтрам папки и Query.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	filter - UUID проекта и optional UUID папки и Query
//
// Что делает функция:
//
//	Передает фильтры в SQLC COUNT и исключает soft-deleted DataView и DataView со soft-deleted Query.
//
// Возвращаемые значения:
//
//	int64 - количество активных DataView
//	error - внутренняя ошибка хранения
func (r *DataViewsRepository) Count(ctx context.Context, filter ports.DataViewsFilter) (count int64, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.data_views.Count")
	defer func() { step.End(err) }()

	count, err = r.queries(ctx).CountDataViews(ctx, sqlc.CountDataViewsParams{
		ProjectID: filter.ProjectID,
		FolderID:  mappers.NullableUUIDToSQLC(filter.FolderID),
		QueryID:   mappers.NullableUUIDToSQLC(filter.QueryID),
	})
	if err != nil {
		r.logger.Error("count data views failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count data views")
	}

	return count, nil
}

func (r *DataViewsRepository) changeRows(ctx context.Context, op, message string, id uuid.UUID, change func(context.Context, uuid.UUID) (int64, error)) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := change(ctx, id)
	if err != nil {
		r.logger.Error(message, zap.Error(err))
		return apperrors.Internal("internal_error", message)
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "data view not found")
	}

	return nil
}

func (r *DataViewsRepository) mapGetError(err error, message string) (*entities.DataView, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("not_found", "data view not found")
	}
	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get data view")
}

func (r *DataViewsRepository) mapWriteError(err error, message string) error {
	r.logger.Error(message, zap.Error(err))
	return mapStorageError(err, dataViewStorageErrorMapping)
}
