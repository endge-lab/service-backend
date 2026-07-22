package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var _ ports.WorkspacesRepository = (*WorkspacesRepository)(nil)

// WorkspacesRepository persists root workspace scopes in PostgreSQL.
type WorkspacesRepository struct{ *baseRepository }

func NewWorkspacesRepository(queries *sqlc.Queries, core *observability.Core, metrics *RepositoryMetrics) *WorkspacesRepository {
	return &WorkspacesRepository{baseRepository: newBaseRepository(queries, core, metrics, "workspaces")}
}

// Create сохраняет новый корневой workspace с полной configuration.
//
// Параметры:
//
//	ctx - контекст выполнения, содержащий при необходимости транзакцию;
//	workspace - доменная сущность с identity, displayName и полной configuration.
//
// Что делает функция:
//
//	Сериализует полную nested configuration в JSONB без разложения по legacy
//	колонкам. Выполняет CreateWorkspace через SQLC и преобразует сохранённую
//	строку обратно в RWorkspace. Ошибки сериализации и десериализации не
//	скрываются пустой configuration; ошибка уникальности identity маппится в
//	безопасный domain conflict. Secret-поля configuration не добавляются в логи
//	или trace fields.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - созданный workspace с ID и timestamps;
//	error - domain error валидации/конфликта либо внутренняя ошибка хранения.
func (r *WorkspacesRepository) Create(ctx context.Context, workspace *entities.RWorkspace) (result *entities.RWorkspace, err error) {
	const op = "repo.workspaces.create"
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	params, err := mappers.CreateWorkspaceParams(workspace)
	if err != nil {
		r.observer.Logger().Error("serialize workspace configuration failed", zap.Error(err))
		return nil, domainerrors.Internal("internal_error", "failed to save workspace")
	}

	value, err := r.queries(ctx).CreateWorkspace(ctx, params)
	if err != nil {
		r.observer.Logger().Error("create workspace failed", zap.Error(err))
		return nil, mapStorageError(err, workspaceStorageErrorMapping)
	}

	return r.mapWorkspace(value)
}

// List возвращает все workspace без фильтрации по пользователю.
//
// Параметры:
//
//	ctx - контекст выполнения, содержащий при необходимости транзакцию.
//
// Что делает функция:
//
//	Выполняет ListWorkspaces, упорядоченный SQLC-запросом по createdAt. Для
//	каждой строки десериализует полную JSONB configuration и формирует доменную
//	сущность. В задаче 04 authentication, memberships и roles отсутствуют,
//	поэтому repository намеренно не применяет фильтр доступа.
//
// Возвращаемые значения:
//
//	[]*entities.RWorkspace - список всех workspace;
//	error - внутренняя ошибка чтения или безопасная ошибка JSONB-маппинга.
func (r *WorkspacesRepository) List(ctx context.Context) (result []*entities.RWorkspace, err error) {
	const op = "repo.workspaces.list"
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	values, err := r.queries(ctx).ListWorkspaces(ctx)
	if err != nil {
		r.observer.Logger().Error("list workspaces failed", zap.Error(err))
		return nil, domainerrors.Internal("internal_error", "failed to list workspaces")
	}

	result = make([]*entities.RWorkspace, 0, len(values))
	for _, value := range values {
		workspace, mapErr := r.mapWorkspace(value)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, workspace)
	}

	return result, nil
}

// GetByIdentity возвращает workspace по его стабильному root identity.
//
// Параметры:
//
//	ctx - контекст выполнения, содержащий при необходимости транзакцию;
//	identity - нормализованный человекочитаемый identity workspace.
//
// Что делает функция:
//
//	Выполняет GetWorkspaceByIdentity через SQLC. Отсутствующая строка
//	преобразуется в безопасную domain not-found ошибку; найденная JSONB
//	configuration десериализуется в полную EndgeConfiguration.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - найденный workspace;
//	error - not_found, ошибка JSONB-маппинга либо внутренняя ошибка хранения.
func (r *WorkspacesRepository) GetByIdentity(ctx context.Context, identity string) (result *entities.RWorkspace, err error) {
	const op = "repo.workspaces.get_by_identity"
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	value, err := r.queries(ctx).GetWorkspaceByIdentity(ctx, identity)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("not_found", "workspace not found")
		}
		r.observer.Logger().Error("get workspace by identity failed", zap.Error(err))
		return nil, domainerrors.Internal("internal_error", "failed to get workspace")
	}

	return r.mapWorkspace(value)
}

// Update сохраняет разрешённые поля workspace и полную замену configuration.
//
// Параметры:
//
//	ctx - контекст выполнения, содержащий при необходимости транзакцию;
//	workspace - уже разрешённая доменная сущность с неизменяемыми ID/identity и
//	обновлёнными displayName и/или полной configuration.
//
// Что делает функция:
//
//	Сериализует переданную полную configuration в JSONB и выполняет
//	UpdateWorkspace по техническому UUID. Repository не выполняет partial JSON
//	merge: usecase передаёт либо сохранённую configuration, либо уже
//	валидированную полную replacement configuration. Отсутствующая запись
//	маппится в not_found, а constraint-ошибки — в безопасные domain errors.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - обновлённый workspace с новым updatedAt;
//	error - not_found, conflict/validation либо внутренняя ошибка хранения.
func (r *WorkspacesRepository) Update(ctx context.Context, workspace *entities.RWorkspace) (result *entities.RWorkspace, err error) {
	const op = "repo.workspaces.update"
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	params, err := mappers.UpdateWorkspaceParams(workspace)
	if err != nil {
		r.observer.Logger().Error("serialize workspace configuration failed", zap.Error(err))
		return nil, domainerrors.Internal("internal_error", "failed to save workspace")
	}

	value, err := r.queries(ctx).UpdateWorkspace(ctx, params)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("not_found", "workspace not found")
		}
		r.observer.Logger().Error("update workspace failed", zap.Error(err))
		return nil, mapStorageError(err, workspaceStorageErrorMapping)
	}

	return r.mapWorkspace(value)
}

func (r *WorkspacesRepository) mapWorkspace(value sqlc.Workspace) (*entities.RWorkspace, error) {
	workspace, err := mappers.Workspace(value)
	if err != nil {
		r.observer.Logger().Error("decode workspace configuration failed", zap.Error(err))
		return nil, domainerrors.Internal("internal_error", "failed to read workspace")
	}

	return workspace, nil
}
