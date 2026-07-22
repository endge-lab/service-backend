package projects

import (
	"context"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const projectOperationTimeout = 15 * time.Second

type Project struct {
	projectRepository ports.ProjectsRepository
	folderRepository  ports.FoldersRepository
	txManager         ports.TxManager
	observer          observability.Observer
}

type ProjectParams struct {
	ProjectRepository ports.ProjectsRepository
	FolderRepository  ports.FoldersRepository
	TxManager         ports.TxManager
	Observability     *observability.Core
	Metrics           *shared.UseCaseMetrics
}

func NewProjectService(params ProjectParams) *Project {
	return &Project{
		projectRepository: params.ProjectRepository,
		folderRepository:  params.FolderRepository,
		txManager:         params.TxManager,
		observer:          params.Observability.For(observability.LayerUseCase, "projects_usecase").WithRecorder(params.Metrics),
	}
}

// Create создает проект и его системные корневые папки.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - данные для создания проекта
//
// Что делает функция:
//
//	Валидирует входные данные и создает проект.
//	В одной транзакции создает root folders для всех поддерживаемых entity types.
//
// Возвращаемые значения:
//
//	*entities.RProject - созданный проект
//	error - ошибка, возникшая при выполнении операции
func (s *Project) Create(ctx context.Context, input CreateProjectInput) (result *entities.RProject, err error) {
	const op = "project.create"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err := normalizeAndValidateCreateProjectInput(&input); err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}
	workspaceID, ok := entities.WorkspaceIDFromContext(ctx)
	if !ok {
		err = apperrors.InvalidInput("workspace_required", "workspace scope is required")
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	exists, err := s.projectRepository.ExistsByIdentity(ctx, input.Identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	if exists {
		err = apperrors.Conflict("identity_conflict", "project identity already exists")
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
		return nil, err
	}

	project := projectFromCreateInput(input, workspaceID)

	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		result, err = s.projectRepository.Create(txCtx, project)
		if err != nil {
			return err
		}

		for _, root := range projectRootFolders(workspaceID, result.ID) {
			if _, err = s.folderRepository.Create(txCtx, root); err != nil {
				observed.Logger().Error(op,
					zap.Error(err),
					zap.String("project_id", result.ID.String()),
					zap.String("root_identity", root.Identity),
					zap.String("entity_type", string(root.EntityType)),
				)
				return err
			}
		}

		return nil
	})
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	observed.AddEvent("project.created",
		attribute.String("project.id", result.ID.String()),
		attribute.String("workspace.id", workspaceID.String()),
		attribute.Int("project.root_count", len(projectRootEntityTypes)),
	)
	observed.Logger().Info("project created with root folders",
		zap.String("project_id", result.ID.String()),
		zap.String("identity", result.Identity),
		zap.Int("root_count", len(projectRootEntityTypes)),
	)
	return result, nil
}

// GetByID возвращает активный проект по техническому UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID проекта
//
// Что делает функция:
//
//	Валидирует UUID и получает проект из repository.
//	Soft-deleted проекты не возвращаются.
//
// Возвращаемые значения:
//
//	*entities.RProject - найденный проект
//	error - ошибка, возникшая при выполнении операции
func (s *Project) GetByID(ctx context.Context, id uuid.UUID) (result *entities.RProject, err error) {
	const op = "project.get_by_id"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("validation_error", "project id is required")
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	result, err = s.projectRepository.GetByID(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("project_id", id.String()))
		return nil, err
	}

	return result, nil
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
//	Нормализует identity и получает проект из repository.
//	Soft-deleted проекты не возвращаются.
//
// Возвращаемые значения:
//
//	*entities.RProject - найденный проект
//	error - ошибка, возникшая при выполнении операции
func (s *Project) GetByIdentity(ctx context.Context, identity string) (result *entities.RProject, err error) {
	const op = "project.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	identity = strings.TrimSpace(identity)

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if identity == "" {
		err = apperrors.InvalidInput("validation_error", "project identity is required")
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	result, err = s.projectRepository.GetByIdentity(ctx, identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return nil, err
	}

	return result, nil
}

// List возвращает список активных проектов.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Получает список проектов из repository.
//	Исключает soft-deleted записи.
//
// Возвращаемые значения:
//
//	[]*entities.RProject - список проектов
//	error - ошибка, возникшая при выполнении операции
func (s *Project) List(ctx context.Context) (result []*entities.RProject, err error) {
	const op = "project.list"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	result, err = s.projectRepository.List(ctx)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	observed.Logger().Debug("projects listed", zap.Int("count", len(result)))
	return result, nil
}

// Update обновляет проект, найденный по identity.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - identity проекта и новые значения редактируемых полей
//
// Что делает функция:
//
//	Находит активный проект по identity.
//	Обновляет displayName, description, active и meta.
//
// Возвращаемые значения:
//
//	*entities.RProject - обновленный проект
//	error - ошибка, возникшая при выполнении операции
func (s *Project) Update(ctx context.Context, input UpdateProjectInput) (result *entities.RProject, err error) {
	const op = "project.update"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err := normalizeAndValidateUpdateProjectInput(&input); err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	current, err := s.projectRepository.GetByIdentity(ctx, input.Identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
		return nil, err
	}

	project := projectFromUpdateInput(current, input)

	result, err = s.projectRepository.Update(ctx, project)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	observed.Logger().Debug("project updated",
		zap.String("project_id", result.ID.String()),
		zap.String("identity", result.Identity),
	)
	return result, nil
}

// SoftDelete выполняет мягкое удаление проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Разрешает identity во внутренний UUID.
//	Заполняет deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Project) SoftDelete(ctx context.Context, identity string) (err error) {
	const op = "project.soft_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	identity = strings.TrimSpace(identity)
	if identity == "" {
		err = apperrors.InvalidInput("validation_error", "project identity is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	err = s.projectRepository.SoftDelete(ctx, project.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	observed.Logger().Debug("project soft deleted",
		zap.String("project_id", project.ID.String()),
		zap.String("identity", identity),
	)
	return nil
}

// Restore восстанавливает мягко удаленный проект.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Находит проект с учетом soft-deleted записей.
//	Очищает deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Project) Restore(ctx context.Context, identity string) (err error) {
	const op = "project.restore"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	identity = strings.TrimSpace(identity)
	if identity == "" {
		err = apperrors.InvalidInput("validation_error", "project identity is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	project, err := s.projectRepository.GetByIdentityIncludingDeleted(ctx, identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	err = s.projectRepository.Restore(ctx, project.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	observed.Logger().Debug("project restored",
		zap.String("project_id", project.ID.String()),
		zap.String("identity", identity),
	)
	return nil
}

// HardDelete физически удаляет проект.
//
// Параметры:
//
//	ctx - контекст выполнения
//	identity - человекочитаемый идентификатор проекта
//
// Что делает функция:
//
//	Разрешает identity во внутренний UUID.
//	Удаляет проект и каскадно удаляет связанные folders.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Project) HardDelete(ctx context.Context, identity string) (err error) {
	const op = "project.hard_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	identity = strings.TrimSpace(identity)
	if identity == "" {
		err = apperrors.InvalidInput("validation_error", "project identity is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	project, err := s.projectRepository.GetByIdentityIncludingDeleted(ctx, identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	err = s.projectRepository.HardDelete(ctx, project.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", identity))
		return err
	}

	observed.Logger().Debug("project hard deleted",
		zap.String("project_id", project.ID.String()),
		zap.String("identity", identity),
	)
	return nil
}

// Count возвращает количество активных проектов.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Подсчитывает проекты без soft-deleted записей.
//
// Возвращаемые значения:
//
//	int64 - количество проектов
//	error - ошибка, возникшая при выполнении операции
func (s *Project) Count(ctx context.Context) (result int64, err error) {
	const op = "project.count"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	result, err = s.projectRepository.Count(ctx)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return 0, err
	}

	observed.Logger().Debug("projects counted", zap.Int64("count", result))
	return result, nil
}

func normalizeAndValidateCreateProjectInput(input *CreateProjectInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Identity == "" {
		return apperrors.InvalidInput("validation_error", "project identity is required")
	}

	if input.DisplayName == "" {
		return apperrors.InvalidInput("validation_error", "project display name is required")
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	return nil
}
