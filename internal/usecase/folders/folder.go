package folders

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const folderOperationTimeout = 15 * time.Second

type Folder struct {
	folderRepository  ports.FoldersRepository
	projectRepository ports.ProjectsRepository
	observed          shared.ObservedUseCase
}

type FolderParams struct {
	FolderRepository  ports.FoldersRepository
	ProjectRepository ports.ProjectsRepository
	Tracer            trace.Tracer
	Logger            *zap.Logger
	Metrics           *shared.UseCaseMetrics
}

func NewFolderService(params FolderParams) *Folder {
	return &Folder{
		folderRepository:  params.FolderRepository,
		projectRepository: params.ProjectRepository,
		observed: shared.NewObservedUseCase(
			params.Tracer,
			params.Logger,
			params.Metrics,
		),
	}
}

// Create создает папку внутри проекта для указанного entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - данные проекта, папки и optional parent identity
//
// Что делает функция:
//
//	Разрешает projectIdentity и parentIdentity во внутренние UUID.
//	Проверяет уникальность identity и создает папку.
//
// Возвращаемые значения:
//
//	*entities.RFolder - созданная папка
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) Create(
	ctx context.Context,
	input CreateFolderInput,
) (result *entities.RFolder, err error) {
	const op = "folder.create"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	exists, err := s.folderRepository.ExistsByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}
	if exists {
		err = apperrors.Conflict("identity_conflict", "folder identity already exists")
		observed.Logger().Warn(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("parent_identity", nullableString(input.ParentIdentity)),
		)
		return nil, err
	}

	folder := &entities.RFolder{
		WorkspaceID: project.WorkspaceID,
		ProjectID:   new(project.ID),
		EntityType:  input.EntityType,
		Identity:    input.Identity,
		DisplayName: input.DisplayName,
		Description: input.Description,
		ParentID:    parentID,
		Meta:        input.Meta,
	}

	result, err = s.folderRepository.Create(ctx, folder)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return nil, err
	}

	observed.Logger().Debug("folder created",
		zap.String("folder_id", result.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", result.Identity),
		zap.String("entity_type", string(result.EntityType)),
	)
	return result, nil
}

// Update обновляет папку внутри проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - identities папки и новые значения редактируемых полей
//
// Что делает функция:
//
//	Разрешает identities и проверяет корректность нового parent.
//	Запрещает перемещение root folder и создание cycle.
//
// Возвращаемые значения:
//
//	*entities.RFolder - обновленная папка
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) Update(
	ctx context.Context,
	input UpdateFolderInput,
) (result *entities.RFolder, err error) {
	const op = "folder.update"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	current, err := s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("parent_identity", nullableString(input.ParentIdentity)),
		)
		return nil, err
	}

	if parentID != nil && *parentID == current.ID {
		return nil, apperrors.InvalidInput(
			"folder_cycle",
			"folder cycle is not allowed",
		)
	}

	if current.IsRoot && !sameNullableUUID(current.ParentID, parentID) {
		err = apperrors.InvalidInput("validation_error", "root folder cannot be moved")
		observed.Logger().Warn(op, zap.Error(err), zap.String("folder_id", current.ID.String()))
		return nil, err
	}
	if err = s.validateNoCycle(ctx, current.ID, parentID); err != nil {
		observed.Logger().Warn(op, zap.Error(err), zap.String("folder_id", current.ID.String()))
		return nil, err
	}

	updated := &entities.RFolder{
		ID:          current.ID,
		WorkspaceID: current.WorkspaceID,
		ProjectID:   current.ProjectID,
		EntityType:  current.EntityType,
		Identity:    current.Identity,
		DisplayName: input.DisplayName,
		Description: input.Description,
		ParentID:    parentID,
		IsRoot:      current.IsRoot,
		IsSystem:    current.IsSystem,
		DeletedAt:   current.DeletedAt,
		Meta:        input.Meta,
		CreatedAt:   current.CreatedAt,
		UpdatedAt:   current.UpdatedAt,
	}

	result, err = s.folderRepository.Update(ctx, updated)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("folder_id", current.ID.String()),
			zap.String("folder_identity", current.Identity),
		)
		return nil, err
	}

	observed.Logger().Debug("folder updated",
		zap.String("folder_id", result.ID.String()),
		zap.String("folder_identity", result.Identity),
	)
	return result, nil
}

// GetByID возвращает активную папку по техническому UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Валидирует UUID и получает папку из repository.
//	Soft-deleted папки не возвращаются.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) GetByID(ctx context.Context, id uuid.UUID) (result *entities.RFolder, err error) {
	const op = "folder.get_by_id"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("validation_error", "folder id is required")
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	result, err = s.folderRepository.GetByID(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", id.String()))
		return nil, err
	}

	return result, nil
}

// GetByIdentity возвращает активную папку по identities и entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity, entityType и folder identity
//
// Что делает функция:
//
//	Разрешает projectIdentity во внутренний UUID.
//	Получает активную папку по составному identity.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) GetByIdentity(
	ctx context.Context,
	input GetFolderInput,
) (result *entities.RFolder, err error) {
	const op = "folder.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.EntityType, &input.Identity); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	result, err = s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}

	return result, nil
}

// List возвращает активные папки проекта для указанного entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity и entityType
//
// Что делает функция:
//
//	Разрешает projectIdentity во внутренний UUID.
//	Возвращает folders без soft-deleted записей.
//
// Возвращаемые значения:
//
//	[]*entities.RFolder - список папок
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) List(
	ctx context.Context,
	input ListFoldersInput,
) (result []*entities.RFolder, err error) {
	const op = "folder.list"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	result, err = s.folderRepository.List(ctx, &project.ID, input.EntityType)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}

	observed.Logger().Debug("folders listed",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("entity_type", string(input.EntityType)),
		zap.Int("count", len(result)),
	)
	return result, nil
}

// SoftDelete выполняет мягкое удаление папки.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity, entityType и folder identity
//
// Что делает функция:
//
//	Разрешает identities во внутренний UUID папки.
//	Заполняет deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) SoftDelete(ctx context.Context, input FolderIdentityInput) (err error) {
	const op = "folder.soft_delete"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, false)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}

	err = s.folderRepository.SoftDelete(ctx, folder.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	observed.Logger().Debug("folder soft deleted", zap.String("folder_id", folder.ID.String()))
	return nil
}

// Restore восстанавливает мягко удаленную папку.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity, entityType и folder identity
//
// Что делает функция:
//
//	Находит папку с учетом soft-deleted записей.
//	Очищает deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) Restore(ctx context.Context, input FolderIdentityInput) (err error) {
	const op = "folder.restore"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}

	err = s.folderRepository.Restore(ctx, folder.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	observed.Logger().Debug("folder restored", zap.String("folder_id", folder.ID.String()))
	return nil
}

// HardDelete физически удаляет папку.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity, entityType и folder identity
//
// Что делает функция:
//
//	Разрешает identities во внутренний UUID папки.
//	Запрещает hard-delete системной root folder.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) HardDelete(ctx context.Context, input FolderIdentityInput) (err error) {
	const op = "folder.hard_delete"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}
	if folder.IsRoot && folder.IsSystem {
		err = apperrors.Conflict(
			"system_folder_delete_forbidden",
			"system root folder cannot be hard deleted",
		)
		observed.Logger().Warn(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	err = s.folderRepository.HardDelete(ctx, folder.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	observed.Logger().Debug("folder hard deleted", zap.String("folder_id", folder.ID.String()))
	return nil
}

// Count возвращает количество активных папок проекта для указанного entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	input - projectIdentity и entityType
//
// Что делает функция:
//
//	Разрешает projectIdentity во внутренний UUID.
//	Подсчитывает folders без soft-deleted записей.
//
// Возвращаемые значения:
//
//	int64 - количество папок
//	error - ошибка, возникшая при выполнении операции
func (s *Folder) Count(
	ctx context.Context,
	input ListFoldersInput,
) (result int64, err error) {
	const op = "folder.count"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return 0, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}

	count, err := s.folderRepository.Count(ctx, &project.ID, input.EntityType)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return 0, err
	}

	observed.Logger().Debug("folders counted",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("entity_type", string(input.EntityType)),
		zap.Int64("count", count),
	)
	return count, nil
}
