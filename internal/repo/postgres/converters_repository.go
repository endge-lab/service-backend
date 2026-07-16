package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var _ ports.ConvertersRepository = (*ConvertersRepository)(nil)

type ConvertersRepository struct{ *baseRepository }

func NewConvertersRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *ConvertersRepository {
	return &ConvertersRepository{baseRepository: newBaseRepository(queries, tracer, logger, "converters")}
}

// Create сохраняет новый конвертер в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) Create(ctx context.Context, converter *entities.RConverter) (result *entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.Create")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).CreateConverter(ctx, mappers.CreateConverterParams(converter))
	if err != nil {
		return nil, r.mapWriteError(err, "create converter failed")
	}
	return mappers.Converter(value), nil
}

// GetByID возвращает активный конвертер по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.GetByID")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetConverterByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get converter by id failed")
	}
	return mappers.Converter(value), nil
}

// GetByIdentity возвращает активный конвертер по identity проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.GetByIdentity")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetConverterByIdentity(ctx, sqlc.GetConverterByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get converter by identity failed")
	}
	return mappers.Converter(value), nil
}

// GetByIdentityIncludingDeleted возвращает конвертер с учетом soft-delete.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.GetByIdentityIncludingDeleted")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetConverterByIdentityIncludingDeleted(ctx, sqlc.GetConverterByIdentityIncludingDeletedParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get converter by identity including deleted failed")
	}
	return mappers.Converter(value), nil
}

// List возвращает активные конвертеры по фильтру.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) List(ctx context.Context, filter ports.ConvertersFilter) (result []*entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.List")
	defer func() { step.End(err) }()
	values, err := r.queries(ctx).ListConverters(ctx, sqlc.ListConvertersParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID)})
	if err != nil {
		r.logger.Error("list converters failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list converters")
	}
	result = make([]*entities.RConverter, 0, len(values))
	for _, value := range values {
		result = append(result, mappers.Converter(value))
	}
	return result, nil
}

// Update обновляет активный конвертер.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) Update(ctx context.Context, converter *entities.RConverter) (result *entities.RConverter, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.Update")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).UpdateConverter(ctx, mappers.UpdateConverterParams(converter))
	if err != nil {
		return r.mapGetError(err, "update converter failed")
	}
	return mappers.Converter(value), nil
}

// SoftDelete выполняет мягкое удаление конвертера.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.converters.SoftDelete", "soft delete converter failed", id, r.queries(ctx).SoftDeleteConverter)
}

// Restore восстанавливает мягко удаленный конвертер.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.converters.Restore", "restore converter failed", id, r.queries(ctx).RestoreConverter)
}

// HardDelete физически удаляет конвертер.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.converters.HardDelete", "hard delete converter failed", id, r.queries(ctx).HardDeleteConverter)
}

// ExistsByIdentity проверяет существование converter identity в проекте.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.ExistsByIdentity")
	defer func() { step.End(err) }()
	exists, err = r.queries(ctx).ExistsConverterByIdentity(ctx, sqlc.ExistsConverterByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		r.logger.Error("exists converter by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check converter identity")
	}
	return exists, nil
}

// Count возвращает количество активных конвертеров по фильтру.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию согласно правилам текущего слоя.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ConvertersRepository) Count(ctx context.Context, filter ports.ConvertersFilter) (count int64, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.converters.Count")
	defer func() { step.End(err) }()
	count, err = r.queries(ctx).CountConverters(ctx, sqlc.CountConvertersParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID)})
	if err != nil {
		r.logger.Error("count converters failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count converters")
	}
	return count, nil
}
func (r *ConvertersRepository) changeRows(ctx context.Context, op, message string, id uuid.UUID, change func(context.Context, uuid.UUID) (int64, error)) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()
	affected, err := change(ctx, id)
	if err != nil {
		r.logger.Error(message, zap.Error(err))
		return apperrors.Internal("internal_error", message)
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "converter not found")
	}
	return nil
}
func (r *ConvertersRepository) mapGetError(err error, message string) (*entities.RConverter, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("not_found", "converter not found")
	}
	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get converter")
}
func (r *ConvertersRepository) mapWriteError(err error, message string) error {
	r.logger.Error(message, zap.Error(err))
	return mapStorageError(err, converterStorageErrorMapping)
}
