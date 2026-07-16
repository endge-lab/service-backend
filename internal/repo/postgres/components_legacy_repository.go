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

var _ ports.ComponentsLegacyRepository = (*ComponentsLegacyRepository)(nil)

type ComponentsLegacyRepository struct{ *baseRepository }

func NewComponentsLegacyRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *ComponentsLegacyRepository {
	return &ComponentsLegacyRepository{baseRepository: newBaseRepository(queries, tracer, logger, "components_legacy")}
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
func (r *ComponentsLegacyRepository) Create(ctx context.Context, component *entities.RComponentLegacy) (result *entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.Create")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).CreateComponentLegacy(ctx, mappers.CreateComponentLegacyParams(component))
	if err != nil {
		return nil, r.mapWriteError(err, "create component failed")
	}
	return mappers.ComponentLegacy(value), nil
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
func (r *ComponentsLegacyRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.GetByID")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentLegacyByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get component by id failed")
	}
	return mappers.ComponentLegacy(value), nil
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
func (r *ComponentsLegacyRepository) GetByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.GetByIdentity")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentLegacyByIdentity(ctx, sqlc.GetComponentLegacyByIdentityParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get component by identity failed")
	}
	return mappers.ComponentLegacy(value), nil
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
func (r *ComponentsLegacyRepository) GetByIdentityIncludingDeleted(ctx context.Context, projectID uuid.UUID, identity string) (result *entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.GetByIdentityIncludingDeleted")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).GetComponentLegacyByIdentityIncludingDeleted(ctx, sqlc.GetComponentLegacyByIdentityIncludingDeletedParams{ProjectID: projectID, Identity: identity})
	if err != nil {
		return r.mapGetError(err, "get component by identity including deleted failed")
	}
	return mappers.ComponentLegacy(value), nil
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
func (r *ComponentsLegacyRepository) List(ctx context.Context, filter ports.ComponentsLegacyFilter) (result []*entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.List")
	defer func() { step.End(err) }()
	values, err := r.queries(ctx).ListComponentsLegacy(ctx, sqlc.ListComponentsLegacyParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID), ComponentType: mappers.NullableTextToSQLC(componentTypeString(filter.ComponentType))})
	if err != nil {
		r.logger.Error("list legacy components failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list legacy components")
	}
	result = make([]*entities.RComponentLegacy, 0, len(values))
	for _, value := range values {
		result = append(result, mappers.ComponentLegacy(value))
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
func (r *ComponentsLegacyRepository) Update(ctx context.Context, component *entities.RComponentLegacy) (result *entities.RComponentLegacy, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.Update")
	defer func() { step.End(err) }()
	value, err := r.queries(ctx).UpdateComponentLegacy(ctx, mappers.UpdateComponentLegacyParams(component))
	if err != nil {
		return r.mapGetError(err, "update component failed")
	}
	return mappers.ComponentLegacy(value), nil
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
func (r *ComponentsLegacyRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components_legacy.SoftDelete", "soft delete component failed", id, r.queries(ctx).SoftDeleteComponentLegacy)
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
func (r *ComponentsLegacyRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components_legacy.Restore", "restore component failed", id, r.queries(ctx).RestoreComponentLegacy)
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
func (r *ComponentsLegacyRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	return r.changeRows(ctx, "repo.components_legacy.HardDelete", "hard delete component failed", id, r.queries(ctx).HardDeleteComponentLegacy)
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
func (r *ComponentsLegacyRepository) ExistsByIdentity(ctx context.Context, projectID uuid.UUID, identity string) (exists bool, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.ExistsByIdentity")
	defer func() { step.End(err) }()
	exists, err = r.queries(ctx).ExistsComponentLegacyByIdentity(ctx, sqlc.ExistsComponentLegacyByIdentityParams{ProjectID: projectID, Identity: identity})
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
func (r *ComponentsLegacyRepository) Count(ctx context.Context, filter ports.ComponentsLegacyFilter) (count int64, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, "repo.components_legacy.Count")
	defer func() { step.End(err) }()
	count, err = r.queries(ctx).CountComponentsLegacy(ctx, sqlc.CountComponentsLegacyParams{ProjectID: filter.ProjectID, FolderID: mappers.NullableUUIDToSQLC(filter.FolderID), ComponentType: mappers.NullableTextToSQLC(componentTypeString(filter.ComponentType))})
	if err != nil {
		r.logger.Error("count legacy components failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count legacy components")
	}
	return count, nil
}

func (r *ComponentsLegacyRepository) changeRows(ctx context.Context, op, message string, id uuid.UUID, change func(context.Context, uuid.UUID) (int64, error)) (err error) {
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

func (r *ComponentsLegacyRepository) mapGetError(err error, message string) (*entities.RComponentLegacy, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("not_found", "component not found")
	}
	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get component")
}

func (r *ComponentsLegacyRepository) mapWriteError(err error, message string) error {
	r.logger.Error(message, zap.Error(err))
	return mapStorageError(err, componentLegacyStorageErrorMapping)
}

func componentTypeString(value *entities.RComponentLegacyType) *string {
	if value == nil {
		return nil
	}
	return new(string(*value))
}
