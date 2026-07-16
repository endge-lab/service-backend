package postgres

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var _ ports.FoldersRepository = (*FoldersRepository)(nil)

type FoldersRepository struct {
	*baseRepository
}

func NewFoldersRepository(
	queries *sqlc.Queries,
	tracer trace.Tracer,
	logger *zap.Logger,
) *FoldersRepository {
	return &FoldersRepository{
		baseRepository: newBaseRepository(queries, tracer, logger, "folders"),
	}
}

// Create сохраняет новую папку в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//	folder - доменная сущность папки для создания
//
// Что делает функция:
//
//	Преобразует entity в sqlc params и вставляет запись folders.
//
// Возвращаемые значения:
//
//	*entities.RFolder - созданная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) Create(
	ctx context.Context,
	folder *entities.RFolder,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.Create"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	created, err := r.queries(ctx).CreateFolder(ctx, mappers.CreateFolderParams(folder))
	if err != nil {
		r.logger.Error("create folder failed", zap.Error(err))
		return nil, mapFolderStorageError(err, "internal_error")
	}

	return mappers.Folder(created), nil
}

// Update обновляет данные папки в базе данных.
//
// Параметры:
//
//	ctx - контекст выполнения
//	folder - доменная сущность папки с обновленными полями
//
// Что делает функция:
//
//	Обновляет редактируемые поля и updatedAt активной папки.
//
// Возвращаемые значения:
//
//	*entities.RFolder - обновленная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) Update(
	ctx context.Context,
	folder *entities.RFolder,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.Update"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	updated, err := r.queries(ctx).UpdateFolder(ctx, mappers.UpdateFolderParams(folder))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, folderNotFoundError()
		}

		r.logger.Error("update folder failed", zap.Error(err))
		return nil, mapFolderStorageError(err, "internal_error")
	}

	return mappers.Folder(updated), nil
}

// GetByID возвращает активную папку по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Ищет папку по UUID и исключает soft-deleted запись.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.GetByID"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get folder by id failed")
	}

	return mappers.Folder(folder), nil
}

// GetByIDIncludingDeleted возвращает папку по UUID с учетом soft-deleted записей.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Ищет папку по UUID без фильтра deletedAt.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) GetByIDIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.GetByIDIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByIDIncludingDeleted(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get folder by id including deleted failed")
	}

	return mappers.Folder(folder), nil
}

// GetByIdentity возвращает активную папку по составному identity.
//
// Параметры:
//
//	ctx - контекст выполнения
//	projectID - UUID проекта
//	entityType - тип сущности папки
//	identity - человекочитаемый идентификатор папки
//
// Что делает функция:
//
//	Ищет активную папку внутри projectID и entityType.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) GetByIdentity(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
	identity string,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.GetByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByProjectEntityIdentity(
		ctx,
		sqlc.GetFolderByProjectEntityIdentityParams{
			ProjectID:  mappers.NullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
			Identity:   identity,
		},
	)
	if err != nil {
		return r.mapGetError(err, "get folder by identity failed")
	}

	return mappers.Folder(folder), nil
}

// GetByIdentityIncludingDeleted возвращает папку по составному identity с учетом soft-deleted записей.
//
// Параметры:
//
//	ctx - контекст выполнения
//	projectID - UUID проекта
//	entityType - тип сущности папки
//	identity - человекочитаемый идентификатор папки
//
// Что делает функция:
//
//	Ищет папку внутри projectID и entityType без фильтра deletedAt.
//
// Возвращаемые значения:
//
//	*entities.RFolder - найденная папка
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) GetByIdentityIncludingDeleted(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
	identity string,
) (result *entities.RFolder, err error) {
	const op = "repo.folders.GetByIdentityIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByProjectEntityIdentityIncludingDeleted(
		ctx,
		sqlc.GetFolderByProjectEntityIdentityIncludingDeletedParams{
			ProjectID:  mappers.NullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
			Identity:   identity,
		},
	)
	if err != nil {
		return r.mapGetError(err, "get folder by identity including deleted failed")
	}

	return mappers.Folder(folder), nil
}

// List возвращает список активных папок проекта для указанного entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	projectID - UUID проекта
//	entityType - тип сущности папки
//
// Что делает функция:
//
//	Выбирает folders внутри projectID и entityType с пустым deletedAt.
//
// Возвращаемые значения:
//
//	[]*entities.RFolder - список папок
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) List(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
) (result []*entities.RFolder, err error) {
	const op = "repo.folders.List"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folders, err := r.queries(ctx).ListFoldersByProjectAndEntityType(
		ctx,
		sqlc.ListFoldersByProjectAndEntityTypeParams{
			ProjectID:  mappers.NullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
		},
	)
	if err != nil {
		r.logger.Error("list folders failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list folders")
	}

	result = make([]*entities.RFolder, 0, len(folders))
	for _, folder := range folders {
		result = append(result, mappers.Folder(folder))
	}

	return result, nil
}

// SoftDelete выполняет мягкое удаление папки по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Заполняет deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.folders.SoftDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := r.queries(ctx).SoftDeleteFolder(ctx, id)
	if err != nil {
		r.logger.Error("soft delete folder failed", zap.Error(err))
		return mapFolderStorageError(err, "internal_error")
	}
	if affected == 0 {
		return folderNotFoundError()
	}

	return nil
}

// Restore восстанавливает мягко удаленную папку по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Очищает deletedAt и обновляет updatedAt.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.folders.Restore"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := r.queries(ctx).RestoreFolder(ctx, id)
	if err != nil {
		r.logger.Error("restore folder failed", zap.Error(err))
		return mapFolderStorageError(err, "internal_error")
	}
	if affected == 0 {
		return folderNotFoundError()
	}

	return nil
}

// HardDelete физически удаляет папку по UUID.
//
// Параметры:
//
//	ctx - контекст выполнения
//	id - UUID папки
//
// Что делает функция:
//
//	Проверяет запрет удаления system root и удаляет запись folders.
//
// Возвращаемые значения:
//
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.folders.HardDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByIDIncludingDeleted(ctx, id)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return folderNotFoundError()
		}
		r.logger.Error("get folder before hard delete failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to get folder")
	}
	if folder.IsRoot && folder.IsSystem {
		return apperrors.Conflict(
			"system_folder_delete_forbidden",
			"system root folder cannot be hard deleted",
		)
	}

	affected, err := r.queries(ctx).HardDeleteFolder(ctx, id)
	if err != nil {
		r.logger.Error("hard delete folder failed", zap.Error(err))
		return mapFolderStorageError(err, "internal_error")
	}
	if affected == 0 {
		return folderNotFoundError()
	}

	return nil
}

// ExistsByIdentity проверяет существование папки с указанным identity.
//
// Параметры:
//
//	ctx - контекст выполнения
//	projectID - UUID проекта
//	entityType - тип сущности папки
//	identity - человекочитаемый идентификатор папки
//
// Что делает функция:
//
//	Проверяет уникальность identity внутри projectID и entityType.
//
// Возвращаемые значения:
//
//	bool - true, если identity уже существует
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) ExistsByIdentity(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
	identity string,
) (result bool, err error) {
	const op = "repo.folders.ExistsByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	exists, err := r.queries(ctx).ExistsFolderByProjectEntityIdentity(
		ctx,
		sqlc.ExistsFolderByProjectEntityIdentityParams{
			ProjectID:  mappers.NullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
			Identity:   identity,
		},
	)
	if err != nil {
		r.logger.Error("exists folder by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check folder identity")
	}

	return exists, nil
}

// Count возвращает количество активных папок проекта для указанного entityType.
//
// Параметры:
//
//	ctx - контекст выполнения
//	projectID - UUID проекта
//	entityType - тип сущности папки
//
// Что делает функция:
//
//	Подсчитывает folders внутри projectID и entityType с пустым deletedAt.
//
// Возвращаемые значения:
//
//	int64 - количество папок
//	error - ошибка, возникшая при выполнении операции
func (r *FoldersRepository) Count(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
) (result int64, err error) {
	const op = "repo.folders.Count"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	count, err := r.queries(ctx).CountFoldersByProjectAndEntityType(
		ctx,
		sqlc.CountFoldersByProjectAndEntityTypeParams{
			ProjectID:  mappers.NullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
		},
	)
	if err != nil {
		r.logger.Error("count folders failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count folders")
	}

	return count, nil
}

func (r *FoldersRepository) mapGetError(err error, message string) (*entities.RFolder, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, folderNotFoundError()
	}

	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get folder")
}

func mapFolderStorageError(err error, fallbackCode string) error {
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) && strings.Contains(pgErr.Message, "folder cycle") {
		return apperrors.Conflict("folder_cycle", "folder cycle is not allowed")
	}
	if stderrors.As(err, &pgErr) && strings.Contains(pgErr.Message, "system root folder cannot be hard-deleted") {
		return apperrors.Conflict(
			"system_folder_delete_forbidden",
			"system root folder cannot be hard deleted",
		)
	}

	return mapStorageError(err, storageErrorMapping{
		identityConstraintNames: []string{
			"folders_project_entity_identity_unique",
			"folders_global_entity_identity_unique",
		},
		identityConflictMessage: "folder identity already exists",
		validationMessage:       "folder data violates a constraint",
		internalCode:            apperrors.Code(fallbackCode),
		internalStorageMessage:  "folder storage operation failed",
	})
}

func folderNotFoundError() error {
	return apperrors.NotFound("not_found", "folder not found")
}
