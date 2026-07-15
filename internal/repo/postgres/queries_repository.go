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

var _ ports.QueriesRepository = (*QueriesRepository)(nil)

type QueriesRepository struct{ *baseRepository }

func NewQueriesRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *QueriesRepository {
	return &QueriesRepository{baseRepository: newBaseRepository(queries, tracer, logger, "queries")}
}

// Create сохраняет новую Query в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	query - Query с уже разрешенными UUID проекта и папки
//
// Что делает функция:
//
//	Преобразует domain entity в SQLC-параметры, выполняет INSERT и маппит сохраненную строку.
//
// Возвращаемые значения:
//
//	*entities.Query - Query с полями, заполненными хранилищем
//	error - storage error, преобразованная в domain error
func (r *QueriesRepository) Create(ctx context.Context, query *entities.Query) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.Create")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).CreateQuery(ctx, mappers.CreateQueryParams(query))
	if err != nil {
		return nil, r.mapWriteError(err, "create query failed")
	}

	return mappers.Query(value), nil
}

// GetByID возвращает активную Query по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID Query
//
// Что делает функция:
//
//	Выполняет SQLC-выборку, исключающую soft-deleted запись, и маппит результат в domain entity.
//
// Возвращаемые значения:
//
//	*entities.Query - найденная активная Query
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.GetByID")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetQueryByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get query by id failed")
	}

	return mappers.Query(value), nil
}

// GetByIDIncludingDeleted возвращает Query по UUID с учетом soft-delete.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID Query
//
// Что делает функция:
//
//	Выполняет SQLC-выборку без фильтра deletedAt и маппит результат в domain entity.
//
// Возвращаемые значения:
//
//	*entities.Query - найденная Query, включая soft-deleted запись
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) GetByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.GetByIDIncludingDeleted")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetQueryByIDIncludingDeleted(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get query by id including deleted failed")
	}

	return mappers.Query(value), nil
}

// GetByIdentity возвращает активную Query по identity внутри проекта.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID проекта-владельца Query
//	identity - человекочитаемый идентификатор Query
//
// Что делает функция:
//
//	Выполняет project-scoped SQLC-выборку и исключает soft-deleted запись.
//
// Возвращаемые значения:
//
//	*entities.Query - найденная активная Query
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.GetByIdentity")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetQueryByIdentity(ctx, sqlc.GetQueryByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get query by identity failed")
	}

	return mappers.Query(value), nil
}

// GetByIdentityIncludingDeleted возвращает Query по identity внутри проекта с учетом soft-delete.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID проекта-владельца Query
//	identity - человекочитаемый идентификатор Query
//
// Что делает функция:
//
//	Получает Query без фильтра deletedAt для restore и hard-delete flow.
//
// Возвращаемые значения:
//
//	*entities.Query - найденная Query, включая soft-deleted запись
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.GetByIdentityIncludingDeleted")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetQueryByIdentityIncludingDeleted(ctx, sqlc.GetQueryByIdentityIncludingDeletedParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get query by identity including deleted failed")
	}

	return mappers.Query(value), nil
}

// List возвращает активные Query проекта по optional фильтрам папки и типа.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	filter - UUID проекта и optional UUID папки и query type
//
// Что делает функция:
//
//	Передает фильтры в SQLC, исключает soft-deleted записи и преобразует все строки в domain entities.
//
// Возвращаемые значения:
//
//	[]*entities.Query - список активных Query
//	error - внутренняя ошибка хранения
func (r *QueriesRepository) List(ctx context.Context, filter ports.QueriesFilter) (result []*entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.List")
	defer func() { step.End(err) }()

	values, err := r.queries(ctx).ListQueries(ctx, sqlc.ListQueriesParams{
		ProjectID: filter.ProjectID,
		FolderID:  mappers.NullableUUIDToSQLC(filter.FolderID),
		QueryType: mappers.NullableTextToSQLC(filter.QueryType),
	})
	if err != nil {
		r.logger.Error("list queries failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list queries")
	}

	result = make([]*entities.Query, 0, len(values))
	for _, value := range values {
		result = append(result, mappers.Query(value))
	}

	return result, nil
}

// Update сохраняет editable payload активной Query.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	query - Query с UUID и обновленными editable полями
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE, который сохраняет identity и обновляет updatedAt на стороне БД.
//
// Возвращаемые значения:
//
//	*entities.Query - обновленная активная Query
//	error - not_found, validation error или внутренняя ошибка хранения
func (r *QueriesRepository) Update(ctx context.Context, query *entities.Query) (result *entities.Query, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.Update")
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).UpdateQuery(ctx, mappers.UpdateQueryParams(query))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "query not found")
		}
		return nil, r.mapWriteError(err, "update query failed")
	}

	return mappers.Query(value), nil
}

// SoftDelete помечает активную Query удаленной по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID активной Query
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE deletedAt и updatedAt; проверяет, что была изменена одна или более строк.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.queries.SoftDelete", "soft delete query failed", id, r.queries(ctx).SoftDeleteQuery)
}

// Restore восстанавливает soft-deleted Query по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID soft-deleted Query
//
// Что делает функция:
//
//	Выполняет SQLC UPDATE, очищающий deletedAt и обновляющий updatedAt.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.queries.Restore", "restore query failed", id, r.queries(ctx).RestoreQuery)
}

// HardDelete физически удаляет Query по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	id - UUID soft-deleted Query
//
// Что делает функция:
//
//	Выполняет DELETE; PostgreSQL каскадно удаляет связанные DataView.
//
// Возвращаемые значения:
//
//	error - not_found или внутренняя ошибка хранения
func (r *QueriesRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.queries.HardDelete", "hard delete query failed", id, r.queries(ctx).HardDeleteQuery)
}

// ExistsByIdentity проверяет наличие Query identity внутри проекта.
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
//	bool - признак существования Query
//	error - внутренняя ошибка хранения
func (r *QueriesRepository) ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.ExistsByIdentity")
	defer func() { step.End(err) }()

	exists, err = r.queries(ctx).ExistsQueryByIdentity(ctx, sqlc.ExistsQueryByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		r.logger.Error("exists query by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check query identity")
	}

	return exists, nil
}

// ExistsActiveByIdentityOutsideProject проверяет наличие активной Query identity в другом проекте.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	projectID - UUID текущего проекта
//	identity - проверяемый identity
//
// Что делает функция:
//
//	Выполняет SQLC EXISTS вне текущего проекта и исключает soft-deleted Query.
//
// Возвращаемые значения:
//
//	bool - признак активной Query с identity в другом проекте
//	error - внутренняя ошибка хранения
func (r *QueriesRepository) ExistsActiveByIdentityOutsideProject(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.ExistsActiveByIdentityOutsideProject")
	defer func() { step.End(err) }()

	exists, err = r.queries(ctx).ExistsActiveQueryByIdentityOutsideProject(ctx, sqlc.ExistsActiveQueryByIdentityOutsideProjectParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		r.logger.Error("exists active query by identity outside project failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check query project")
	}

	return exists, nil
}

// Count возвращает количество активных Query проекта по optional фильтрам папки и типа.
//
// Параметры:
//
//	ctx - контекст выполнения SQL-операции
//	filter - UUID проекта и optional UUID папки и query type
//
// Что делает функция:
//
//	Передает фильтры в SQLC COUNT и исключает soft-deleted записи.
//
// Возвращаемые значения:
//
//	int64 - количество активных Query
//	error - внутренняя ошибка хранения
func (r *QueriesRepository) Count(ctx context.Context, filter ports.QueriesFilter) (count int64, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.queries.Count")
	defer func() { step.End(err) }()

	count, err = r.queries(ctx).CountQueries(ctx, sqlc.CountQueriesParams{
		ProjectID: filter.ProjectID,
		FolderID:  mappers.NullableUUIDToSQLC(filter.FolderID),
		QueryType: mappers.NullableTextToSQLC(filter.QueryType),
	})
	if err != nil {
		r.logger.Error("count queries failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count queries")
	}

	return count, nil
}

func (r *QueriesRepository) changeRows(ctx context.Context, op, message string, id uuid.UUID, change func(context.Context, uuid.UUID) (int64, error)) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := change(ctx, id)
	if err != nil {
		r.logger.Error(message, zap.Error(err))
		return apperrors.Internal("internal_error", message)
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "query not found")
	}

	return nil
}

func (r *QueriesRepository) mapGetError(err error, message string) (*entities.Query, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("not_found", "query not found")
	}
	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get query")
}

func (r *QueriesRepository) mapWriteError(err error, message string) error {
	r.logger.Error(message, zap.Error(err))
	return mapStorageError(err, queryStorageErrorMapping)
}
