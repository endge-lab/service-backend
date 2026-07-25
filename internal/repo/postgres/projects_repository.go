package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type ProjectsRepository struct {
	*baseRepository
}

func NewProjectsRepository(
	queries *sqlc.Queries,
	core *observability.Core,
	metrics *RepositoryMetrics,
) *ProjectsRepository {
	return &ProjectsRepository{
		baseRepository: newBaseRepository(queries, core, metrics, "projects"),
	}
}

// Create сохраняет новый проект в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//	project - доменная сущность проекта для создания
//
// Что делает функция:
//
//	Преобразует entity в sqlc params и вставляет запись projects.
//
// Возвращаемые значения:
//
//	*entities.RProject - созданный проект
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) Create(ctx context.Context, project *entities.RProject) (result *entities.RProject, err error) {
	const op = "repo.projects.Create"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	if _, err = requireEntityWorkspace(ctx, project.WorkspaceID); err != nil {
		return nil, err
	}

	created, err := r.queries(ctx).CreateProject(ctx, mappers.CreateProjectParams(project))
	if err != nil {
		r.observer.Logger().Error("create project failed", zap.Error(err))
		return nil, mapProjectWriteError(err)
	}

	return mappers.Project(created), nil
}

// GetByID возвращает активный проект по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID проекта
//
// Что делает функция:
//
//	Ищет проект по UUID и исключает soft-deleted запись.
//
// Возвращаемые значения:
//
//	*entities.RProject - найденный проект
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.RProject, err error) {
	const op = "repo.projects.GetByID"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	project, err := r.queries(ctx).GetProjectByID(ctx, sqlc.GetProjectByIDParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.observer.Logger().Error("get project by id failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mappers.Project(project), nil
}

// GetByIdentity возвращает активный проект по identity.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Ищет проект по identity и исключает soft-deleted запись.
//
// Возвращаемые значения:
//
//	*entities.RProject - найденный проект
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) GetByIdentity(ctx context.Context, identity string) (result *entities.RProject, err error) {
	const op = "repo.projects.GetByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	project, err := r.queries(ctx).GetProjectByIdentity(ctx, sqlc.GetProjectByIdentityParams{WorkspaceID: workspaceID, Identity: identity})
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.observer.Logger().Error("get project by identity failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mappers.Project(project), nil
}

// GetByIdentityIncludingDeleted возвращает проект по identity с учетом soft-deleted записей.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Ищет проект по identity без фильтра deletedAt.
//
// Возвращаемые значения:
//
//	*entities.RProject - найденный проект
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) GetByIdentityIncludingDeleted(
	ctx context.Context,
	identity string,
) (result *entities.RProject, err error) {
	const op = "repo.projects.GetByIdentityIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	project, err := r.queries(ctx).GetProjectByIdentityIncludingDeleted(ctx, sqlc.GetProjectByIdentityIncludingDeletedParams{WorkspaceID: workspaceID, Identity: identity})
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.observer.Logger().Error("get project by identity including deleted failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mappers.Project(project), nil
}

// List возвращает список активных проектов.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Выбирает projects с пустым deletedAt.
//
// Возвращаемые значения:
//
//	[]*entities.RProject - список проектов
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) List(ctx context.Context) (result []*entities.RProject, err error) {
	const op = "repo.projects.List"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	projects, err := r.queries(ctx).ListProjects(ctx, workspaceID)
	if err != nil {
		r.observer.Logger().Error("list projects failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list projects")
	}

	result = make([]*entities.RProject, 0, len(projects))
	for _, project := range projects {
		result = append(result, mappers.Project(project))
	}

	return result, nil
}

// Update обновляет данные проекта в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//	project - доменная сущность проекта с обновленными полями
//
// Что делает функция:
//
//	Обновляет редактируемые поля и updatedAt активного проекта.
//
// Возвращаемые значения:
//
//	*entities.RProject - обновленный проект
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) Update(ctx context.Context, project *entities.RProject) (result *entities.RProject, err error) {
	const op = "repo.projects.Update"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	if _, err = requireEntityWorkspace(ctx, project.WorkspaceID); err != nil {
		return nil, err
	}

	updated, err := r.queries(ctx).UpdateProject(ctx, mappers.UpdateProjectParams(project))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.observer.Logger().Error("update project failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to update project")
	}

	return mappers.Project(updated), nil
}

// SoftDelete выполняет мягкое удаление проекта по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID проекта
//
// Что делает функция:
//
//	Заполняет deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.SoftDelete"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}

	affected, err := r.queries(ctx).SoftDeleteProject(ctx, sqlc.SoftDeleteProjectParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		r.observer.Logger().Error("soft delete project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to delete project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

// Restore восстанавливает мягко удаленный проект по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID проекта
//
// Что делает функция:
//
//	Очищает deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.Restore"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}

	affected, err := r.queries(ctx).RestoreProject(ctx, sqlc.RestoreProjectParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		r.observer.Logger().Error("restore project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to restore project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

// HardDelete физически удаляет проект по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID проекта
//
// Что делает функция:
//
//	Удаляет запись projects и запускает каскадное удаление folders.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.HardDelete"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}

	affected, err := r.queries(ctx).HardDeleteProject(ctx, sqlc.HardDeleteProjectParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		r.observer.Logger().Error("hard delete project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to delete project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

// ExistsByIdentity проверяет существование проекта с указанным identity.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Проверяет глобальную уникальность identity с учетом soft-deleted записей.
//
// Возвращаемые значения:
//
//	bool - true, если identity уже существует
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) ExistsByIdentity(ctx context.Context, identity string) (result bool, err error) {
	const op = "repo.projects.ExistsByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	exists, err := r.queries(ctx).ExistsProjectByIdentity(ctx, sqlc.ExistsProjectByIdentityParams{WorkspaceID: workspaceID, Identity: identity})
	if err != nil {
		r.observer.Logger().Error("exists project by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check project identity")
	}

	return exists, nil
}

// Count возвращает количество активных проектов.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Подсчитывает projects с пустым deletedAt.
//
// Возвращаемые значения:
//
//	int64 - количество проектов
//	error - ошибка, возникшая при выполнении операции
func (r *ProjectsRepository) Count(ctx context.Context) (result int64, err error) {
	const op = "repo.projects.Count"

	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), op)
	defer func() { step.End(err) }()
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return 0, err
	}

	count, err := r.queries(ctx).CountProject(ctx, workspaceID)
	if err != nil {
		r.observer.Logger().Error("count projects failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count projects")
	}

	return count, nil
}

func mapProjectWriteError(err error) error {
	return mapStorageError(err, projectStorageErrorMapping)
}
