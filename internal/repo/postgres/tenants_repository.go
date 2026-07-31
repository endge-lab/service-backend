package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var _ ports.TenantsRepository = (*TenantsRepository)(nil)

// TenantsRepository persists workspace-scoped tenants in PostgreSQL.
type TenantsRepository struct{ *baseRepository }

// NewTenantsRepository создаёт PostgreSQL-реализацию ports.TenantsRepository.
//
// Параметры:
//
//	queries - сгенерированные SQLC queries, поддерживающие tx из ctx;
//	core и metrics - repository-level trace, structured logs и duration metrics.
//
// Возвращает repository без request state. Workspace boundary определяется
// отдельно для каждого вызова через entities.WorkspaceIDFromContext.
func NewTenantsRepository(queries *sqlc.Queries, core *observability.Core, metrics *RepositoryMetrics) *TenantsRepository {
	return &TenantsRepository{baseRepository: newBaseRepository(queries, core, metrics, "tenants")}
}

// Create сохраняет новый tenant в workspace из ctx.
//
// Параметры:
//
//	ctx - контекст с WorkspaceScope и, при необходимости, активной транзакцией;
//	tenant - доменная сущность с workspaceID, identity, code, contribution
//	и опциональным техническим FolderID.
//
// Что делает функция:
//
//	Проверяет соответствие tenant.WorkspaceID request scope. Проверяет, что
//	явный FolderID принадлежит тому же workspace и имеет entity_type tenants;
//	при nil FolderID берёт root-tenants. Сериализует configuration contribution
//	в JSONB, выполняет INSERT ... RETURNING и преобразует результат в RTenant.
//
// Возвращаемые значения:
//
//	*entities.RTenant - сохранённая запись с серверными ID и timestamps;
//	error - workspace scope, folder, conflict, JSONB или storage error.
func (r *TenantsRepository) Create(ctx context.Context, tenant *entities.RTenant) (result *entities.RTenant, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.tenants.create")
	defer func() { step.End(err) }()

	if tenant == nil {
		return nil, apperrors.InvalidInput("tenant_required", "tenant is required")
	}
	if _, err = requireEntityWorkspace(ctx, tenant.WorkspaceID); err != nil {
		return nil, err
	}

	persisted := *tenant
	if persisted.FolderID, err = r.resolveFolderID(ctx, persisted.WorkspaceID, tenant.FolderID); err != nil {
		return nil, err
	}
	params, err := mappers.CreateTenantParams(&persisted)
	if err != nil {
		r.observer.Logger().Error("serialize tenant configuration failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to save tenant")
	}

	value, err := r.queries(ctx).CreateTenant(ctx, params)
	if err != nil {
		r.observer.Logger().Error("create tenant failed", zap.Error(err))
		return nil, mapTenantWriteError(err)
	}

	return r.mapTenant(value)
}

// List читает tenants только из workspace, определённого в ctx.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	filter - optional FolderID, предварительно разрешённый usecase-слоем.
//
// Что делает функция:
//
//	Передаёт workspaceID и folder filter в SQLC query. Для каждой строки
//	десериализует JSONB contribution; effective configuration не вычисляет.
//
// Возвращаемые значения:
//
//	[]*entities.RTenant - список tenants текущего workspace;
//	error - workspace_required, JSONB decode или storage error.
func (r *TenantsRepository) List(ctx context.Context, filter ports.TenantsFilter) (result []*entities.RTenant, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.tenants.list")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	values, err := r.queries(ctx).ListTenants(ctx, sqlc.ListTenantsParams{
		WorkspaceID: workspaceID,
		FolderID:    mappers.NullableUUIDToSQLC(filter.FolderID),
	})
	if err != nil {
		r.observer.Logger().Error("list tenants failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list tenants")
	}

	result = make([]*entities.RTenant, 0, len(values))
	for _, value := range values {
		tenant, mapErr := r.mapTenant(value)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, tenant)
	}
	return result, nil
}

// GetByIdentity возвращает tenant по stable identity из workspace в ctx.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	identity - normalized public tenant identity.
//
// Что делает функция:
//
//	Выполняет SELECT с обязательными workspace_id и identity, поэтому строка
//	из другого workspace не может попасть в результат. Отсутствие строки
//	конвертируется в безопасный domain not_found.
//
// Возвращаемые значения:
//
//	*entities.RTenant - tenant с десериализованной contribution;
//	error - workspace_required, not_found, JSONB decode или storage error.
func (r *TenantsRepository) GetByIdentity(ctx context.Context, identity string) (result *entities.RTenant, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.tenants.get_by_identity")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	value, err := r.queries(ctx).GetTenantByIdentity(ctx, sqlc.GetTenantByIdentityParams{
		WorkspaceID: workspaceID,
		Identity:    identity,
	})
	if err != nil {
		return r.mapTenantGetError(err, "get tenant by identity failed")
	}

	return r.mapTenant(value)
}

// Update сохраняет полностью собранное новое состояние tenant.
//
// Параметры:
//
//	ctx - контекст с WorkspaceScope и, при необходимости, транзакцией;
//	tenant - entity, в которой usecase уже сохранил immutable поля и применил patch.
//
// Что делает функция:
//
//	Проверяет соответствие workspaceID scope, повторно валидирует FolderID
//	внутри workspace и сериализует contribution. SQL UPDATE ищет запись по
//	workspace_id + identity, поэтому не обновит tenant соседнего workspace.
//
// Возвращаемые значения:
//
//	*entities.RTenant - обновлённая строка;
//	error - workspace scope, folder, not_found, conflict, JSONB или storage error.
func (r *TenantsRepository) Update(ctx context.Context, tenant *entities.RTenant) (result *entities.RTenant, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.tenants.update")
	defer func() { step.End(err) }()

	if tenant == nil {
		return nil, apperrors.InvalidInput("tenant_required", "tenant is required")
	}
	if _, err = requireEntityWorkspace(ctx, tenant.WorkspaceID); err != nil {
		return nil, err
	}

	persisted := *tenant
	if persisted.FolderID, err = r.resolveFolderID(ctx, persisted.WorkspaceID, tenant.FolderID); err != nil {
		return nil, err
	}
	params, err := mappers.UpdateTenantParams(&persisted)
	if err != nil {
		r.observer.Logger().Error("serialize tenant configuration failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to save tenant")
	}

	value, err := r.queries(ctx).UpdateTenant(ctx, params)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, tenantNotFoundError()
		}
		r.observer.Logger().Error("update tenant failed", zap.Error(err))
		return nil, mapTenantWriteError(err)
	}

	return r.mapTenant(value)
}

// HardDelete физически удаляет tenant по identity внутри workspace из ctx.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	identity - stable tenant identity.
//
// Что делает функция:
//
//	Выполняет scoped DELETE по workspace_id + identity. Нулевое количество
//	изменённых строк возвращается как not_found, включая случай identity из
//	другого workspace.
//
// Возвращаемые значения:
//
//	error - workspace_required, not_found или storage error.
func (r *TenantsRepository) HardDelete(ctx context.Context, identity string) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.tenants.hard_delete")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}

	affected, err := r.queries(ctx).HardDeleteTenant(ctx, sqlc.HardDeleteTenantParams{
		WorkspaceID: workspaceID,
		Identity:    identity,
	})
	if err != nil {
		r.observer.Logger().Error("hard delete tenant failed", zap.Error(err))
		return mapTenantDeleteError(err)
	}
	if affected == 0 {
		return tenantNotFoundError()
	}
	return nil
}

// resolveFolderID verifies an explicit tenant folder in workspaceID or returns
// the workspace's system root-tenants folder for an omitted folder.
func (r *TenantsRepository) resolveFolderID(ctx context.Context, workspaceID uuid.UUID, requested *uuid.UUID) (*uuid.UUID, error) {
	if requested != nil {
		folderID, err := r.queries(ctx).GetTenantFolderByID(ctx, sqlc.GetTenantFolderByIDParams{
			ID:          *requested,
			WorkspaceID: workspaceID,
		})
		if err == nil {
			return &folderID, nil
		}
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("folder_not_found", "tenant folder not found")
		}
		r.observer.Logger().Error("resolve tenant folder failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to resolve tenant folder")
	}

	folderID, err := r.queries(ctx).GetTenantRootFolder(ctx, workspaceID)
	if err == nil {
		return &folderID, nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.Internal("tenant_root_folder_missing", "tenant root folder is not provisioned")
	}
	r.observer.Logger().Error("resolve tenant root folder failed", zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to resolve tenant root folder")
}

// mapTenant decodes the stored JSONB configuration contribution into the
// domain entity and keeps malformed persisted JSON as an internal error.
func (r *TenantsRepository) mapTenant(value sqlc.Tenant) (*entities.RTenant, error) {
	tenant, err := mappers.Tenant(value)
	if err != nil {
		r.observer.Logger().Error("decode tenant configuration failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to read tenant")
	}
	return tenant, nil
}

// mapTenantGetError converts scoped lookup failures into safe domain errors.
func (r *TenantsRepository) mapTenantGetError(err error, message string) (*entities.RTenant, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, tenantNotFoundError()
	}
	r.observer.Logger().Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get tenant")
}

func mapTenantWriteError(err error) error {
	info := classifyPostgresError(err)
	if info.kind == postgresErrorUniqueViolation {
		switch info.constraintName {
		case "tenants_workspace_identity_unique":
			return apperrors.Conflict("tenant_identity_conflict", "tenant identity already exists")
		case "tenants_workspace_code_unique":
			return apperrors.Conflict("tenant_code_conflict", "tenant code already exists")
		}
	}
	return mapStorageError(err, tenantStorageErrorMapping)
}

func mapTenantDeleteError(err error) error {
	if classifyPostgresError(err).kind == postgresErrorForeignKeyViolation {
		return apperrors.Conflict("tenant_in_use", "tenant is used by persisted records")
	}
	return mapTenantWriteError(err)
}

func tenantNotFoundError() error {
	return apperrors.NotFound("tenant_not_found", "tenant not found")
}
