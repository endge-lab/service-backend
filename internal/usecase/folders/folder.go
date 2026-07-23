package folders

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const folderOperationTimeout = 15 * time.Second

type Folder struct {
	folderRepository  ports.FoldersRepository
	projectRepository ports.ProjectsRepository
	observer          observability.Observer
}

type FolderParams struct {
	FolderRepository  ports.FoldersRepository
	ProjectRepository ports.ProjectsRepository
	Observability     *observability.Core
	Metrics           *shared.UseCaseMetrics
}

func NewFolderService(params FolderParams) *Folder {
	return &Folder{
		folderRepository:  params.FolderRepository,
		projectRepository: params.ProjectRepository,
		observer:          params.Observability.For(observability.LayerUseCase, "folders_usecase").WithRecorder(params.Metrics),
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "folder create input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for folder create", nil,
		zap.String("project_id", project.ID.String()),
		zap.String("project_identity", project.Identity),
	)

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
	observed.RecordStep(op+".identity_available", "folder identity availability confirmed", nil,
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("parent_identity", nullableString(input.ParentIdentity)),
		)
		return nil, err
	}
	observed.RecordStep(op+".parent_resolved", "folder parent resolved", nil,
		zap.String("parent_identity", nullableString(input.ParentIdentity)),
	)

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

	observed.RecordStep(op+".persisted", "folder created", nil,
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "folder update input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for folder update", nil,
		zap.String("project_id", project.ID.String()),
		zap.String("project_identity", project.Identity),
	)

	current, err := s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}
	observed.RecordStep(op+".current_resolved", "folder resolved for update", nil,
		zap.String("folder_id", current.ID.String()),
		zap.String("folder_identity", current.Identity),
	)

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("parent_identity", nullableString(input.ParentIdentity)),
		)
		return nil, err
	}
	observed.RecordStep(op+".parent_resolved", "folder parent resolved for update", nil,
		zap.String("parent_identity", nullableString(input.ParentIdentity)),
	)

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
	observed.RecordStep(op+".hierarchy_validated", "folder hierarchy validated", nil,
		zap.String("folder_id", current.ID.String()),
	)

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

	observed.RecordStep(op+".persisted", "folder updated", nil,
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("validation_error", "folder id is required")
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "folder identifier validated", nil, zap.String("folder_id", id.String()))

	result, err = s.folderRepository.GetByID(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", id.String()))
		return nil, err
	}
	observed.RecordStep(op+".result_loaded", "folder retrieved", nil, zap.String("folder_id", result.ID.String()), zap.String("folder_identity", result.Identity))

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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.EntityType, &input.Identity); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "folder identity input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for folder retrieval", nil,
		zap.String("project_id", project.ID.String()),
		zap.String("project_identity", project.Identity),
	)

	result, err = s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("folder_identity", input.Identity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}
	observed.RecordStep(op+".result_loaded", "folder retrieved", nil, zap.String("folder_id", result.ID.String()), zap.String("folder_identity", result.Identity), zap.String("project_id", project.ID.String()))

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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "folder list input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("entity_type", string(input.EntityType)),
	)

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for folder list", nil,
		zap.String("project_id", project.ID.String()),
		zap.String("project_identity", project.Identity),
	)

	result, err = s.folderRepository.List(ctx, &project.ID, input.EntityType)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return nil, err
	}

	observed.RecordStep(op+".result_loaded", "folders listed", nil,
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "folder state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	folder, err := s.resolveFolder(ctx, &input, false)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "folder resolved for soft delete", nil,
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
	)

	err = s.folderRepository.SoftDelete(ctx, folder.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	observed.RecordStep(op+".persisted", "folder soft deleted", nil, zap.String("folder_id", folder.ID.String()))
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "folder state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "deleted folder resolved for restore", nil,
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
	)

	err = s.folderRepository.Restore(ctx, folder.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("folder_id", folder.ID.String()))
		return err
	}

	observed.RecordStep(op+".persisted", "folder restored", nil, zap.String("folder_id", folder.ID.String()))
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "folder state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", input.Identity),
		zap.String("entity_type", string(input.EntityType)),
	)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.Identity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "deleted folder resolved for hard delete", nil,
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
	)
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

	observed.RecordStep(op+".persisted", "folder hard deleted", nil, zap.String("folder_id", folder.ID.String()))
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

	ctx, observed := s.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return 0, err
	}
	observed.RecordStep(op+".input_validated", "folder count input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("entity_type", string(input.EntityType)),
	)

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for folder count", nil,
		zap.String("project_id", project.ID.String()),
		zap.String("project_identity", project.Identity),
	)

	count, err := s.folderRepository.Count(ctx, &project.ID, input.EntityType)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("entity_type", string(input.EntityType)),
		)
		return 0, err
	}

	observed.RecordStep(op+".result_loaded", "folders counted", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("entity_type", string(input.EntityType)),
		zap.Int64("count", count),
	)
	return count, nil
}
