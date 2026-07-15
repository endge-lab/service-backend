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

var _ ports.ComponentsRepository = (*ComponentsRepository)(nil)

type ComponentsRepository struct{ *baseRepository }

func NewComponentsRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *ComponentsRepository {
	return &ComponentsRepository{baseRepository: newBaseRepository(queries, tracer, logger, "components")}
}

// Create сохраняет новый компонент в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) Create(ctx context.Context, component *entities.Component) (result *entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.Create")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).CreateComponent(ctx, mappers.CreateComponentParams(component))
	if err != nil {
		return nil, r.mapWriteError(err, "create component failed")
	}
	return mappers.Component(value), nil
}

// GetByID возвращает активный компонент по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.GetByID")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get component by id failed")
	}
	return mappers.Component(value), nil
}

// GetByIdentity возвращает активный компонент по identity проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.GetByIdentity")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentByIdentity(ctx, sqlc.GetComponentByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get component by identity failed")
	}
	return mappers.Component(value), nil
}

// GetByIdentityIncludingDeleted возвращает компонент с учетом soft-delete.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.GetByIdentityIncludingDeleted")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentByIdentityIncludingDeleted(ctx, sqlc.GetComponentByIdentityIncludingDeletedParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get component by identity including deleted failed")
	}
	return mappers.Component(value), nil
}

// List возвращает активные компоненты по фильтру.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) List(ctx context.Context, filter ports.ComponentsFilter) (result []*entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.List")
	defer func() { step.End(err) }()
	values, err := r.queries(ctx).ListComponents(ctx, sqlc.ListComponentsParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID), ComponentType: mappers.NullableTextToSQLC(componentTypeString(filter.ComponentType))})
	if err != nil {
		r.logger.Error("list components failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list components")
	}
	result = make([]*entities.Component, 0, len(values))
	for _, value := range values {
		result = append(result, mappers.Component(value))
	}
	return result, nil
}

// Update обновляет активный компонент.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) Update(ctx context.Context, component *entities.Component) (result *entities.Component, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.Update")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).UpdateComponent(ctx, mappers.UpdateComponentParams(component))
	if err != nil {
		return r.mapGetError(err, "update component failed")
	}
	return mappers.Component(value), nil
}

// SoftDelete выполняет мягкое удаление компонента.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components.SoftDelete", "soft delete component failed", id, r.queries(ctx).SoftDeleteComponent)
}

// Restore восстанавливает мягко удаленный компонент.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components.Restore", "restore component failed", id, r.queries(ctx).RestoreComponent)
}

// HardDelete физически удаляет компонент.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components.HardDelete", "hard delete component failed", id, r.queries(ctx).HardDeleteComponent)
}

// ExistsByIdentity проверяет существование component identity в проекте.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.ExistsByIdentity")
	defer func() { step.End(err) }()
	exists, err = r.queries(ctx).ExistsComponentByIdentity(ctx, sqlc.ExistsComponentByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		r.logger.Error("exists component by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check component identity")
	}
	return exists, nil
}

// Count возвращает количество активных компонентов по фильтру.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выполняет операцию хранения данных компонента.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (r *ComponentsRepository) Count(ctx context.Context, filter ports.ComponentsFilter) (count int64, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components.Count")
	defer func() { step.End(err) }()
	count, err = r.queries(ctx).CountComponents(ctx, sqlc.CountComponentsParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID), ComponentType: mappers.NullableTextToSQLC(componentTypeString(filter.ComponentType))})
	if err != nil {
		r.logger.Error("count components failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count components")
	}
	return count, nil
}

func (r *ComponentsRepository) changeRows(ctx context.Context, op, message string, id uuid.UUID, change func(context.Context, uuid.UUID) (int64, error)) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()
	affected, err := change(ctx, id)
	if err != nil {
		r.logger.Error(message, zap.Error(err))
		return apperrors.Internal("internal_error", message)
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "component not found")
	}
	return nil
}

func (r *ComponentsRepository) mapGetError(err error, message string) (*entities.Component, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("not_found", "component not found")
	}
	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get component")
}

func (r *ComponentsRepository) mapWriteError(err error, message string) error {
	r.logger.Error(message, zap.Error(err))
	return mapStorageError(err, componentStorageErrorMapping)
}

func componentTypeString(value *entities.ComponentType) *string {
	if value == nil {
		return nil
	}
	return new(string(*value))
}
