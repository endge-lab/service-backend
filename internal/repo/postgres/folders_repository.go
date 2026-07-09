package postgres

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
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

func (r *FoldersRepository) Create(
	ctx context.Context,
	folder *entities.Folder,
) (result *entities.Folder, err error) {
	const op = "repo.folders.Create"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	created, err := r.queries(ctx).CreateFolder(ctx, mapCreateFolderParams(folder))
	if err != nil {
		r.logger.Error("create folder failed", zap.Error(err))
		return nil, mapFolderStorageError(err, "internal_error")
	}

	return mapFolder(created), nil
}

func (r *FoldersRepository) Update(
	ctx context.Context,
	folder *entities.Folder,
) (result *entities.Folder, err error) {
	const op = "repo.folders.Update"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	updated, err := r.queries(ctx).UpdateFolder(ctx, mapUpdateFolderParams(folder))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, folderNotFoundError()
		}

		r.logger.Error("update folder failed", zap.Error(err))
		return nil, mapFolderStorageError(err, "internal_error")
	}

	return mapFolder(updated), nil
}

func (r *FoldersRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (result *entities.Folder, err error) {
	const op = "repo.folders.GetByID"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByID(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get folder by id failed")
	}

	return mapFolder(folder), nil
}

func (r *FoldersRepository) GetByIDIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (result *entities.Folder, err error) {
	const op = "repo.folders.GetByIDIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByIDIncludingDeleted(ctx, id)
	if err != nil {
		return r.mapGetError(err, "get folder by id including deleted failed")
	}

	return mapFolder(folder), nil
}

func (r *FoldersRepository) GetByIdentity(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
	identity string,
) (result *entities.Folder, err error) {
	const op = "repo.folders.GetByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByProjectEntityIdentity(
		ctx,
		sqlc.GetFolderByProjectEntityIdentityParams{
			ProjectID:  mapNullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
			Identity:   identity,
		},
	)
	if err != nil {
		return r.mapGetError(err, "get folder by identity failed")
	}

	return mapFolder(folder), nil
}

func (r *FoldersRepository) GetByIdentityIncludingDeleted(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
	identity string,
) (result *entities.Folder, err error) {
	const op = "repo.folders.GetByIdentityIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folder, err := r.queries(ctx).GetFolderByProjectEntityIdentityIncludingDeleted(
		ctx,
		sqlc.GetFolderByProjectEntityIdentityIncludingDeletedParams{
			ProjectID:  mapNullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
			Identity:   identity,
		},
	)
	if err != nil {
		return r.mapGetError(err, "get folder by identity including deleted failed")
	}

	return mapFolder(folder), nil
}

func (r *FoldersRepository) List(
	ctx context.Context,
	projectID *uuid.UUID,
	entityType entities.FolderEntityType,
) (result []*entities.Folder, err error) {
	const op = "repo.folders.List"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	folders, err := r.queries(ctx).ListFoldersByProjectAndEntityType(
		ctx,
		sqlc.ListFoldersByProjectAndEntityTypeParams{
			ProjectID:  mapNullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
		},
	)
	if err != nil {
		r.logger.Error("list folders failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list folders")
	}

	result = make([]*entities.Folder, 0, len(folders))
	for _, folder := range folders {
		result = append(result, mapFolder(folder))
	}

	return result, nil
}

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
			ProjectID:  mapNullableUUIDToSQLC(projectID),
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
			ProjectID:  mapNullableUUIDToSQLC(projectID),
			EntityType: string(entityType),
		},
	)
	if err != nil {
		r.logger.Error("count folders failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count folders")
	}

	return count, nil
}

func (r *FoldersRepository) mapGetError(err error, message string) (*entities.Folder, error) {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, folderNotFoundError()
	}

	r.logger.Error(message, zap.Error(err))
	return nil, apperrors.Internal("internal_error", "failed to get folder")
}

func mapFolderStorageError(err error, fallbackCode string) error {
	var pgErr *pgconn.PgError
	if !stderrors.As(err, &pgErr) {
		return apperrors.Internal(apperrors.Code(fallbackCode), "folder storage operation failed")
	}

	switch {
	case pgErr.Code == postgresUniqueViolation &&
		(pgErr.ConstraintName == "folders_project_entity_identity_unique" ||
			pgErr.ConstraintName == "folders_global_entity_identity_unique"):
		return apperrors.Conflict("identity_conflict", "folder identity already exists")
	case strings.Contains(pgErr.Message, "folder cycle"):
		return apperrors.Conflict("folder_cycle", "folder cycle is not allowed")
	case strings.Contains(pgErr.Message, "system root folder cannot be hard-deleted"):
		return apperrors.Conflict(
			"system_folder_delete_forbidden",
			"system root folder cannot be hard deleted",
		)
	case pgErr.Code == postgresForeignKeyViolation || pgErr.Code == postgresCheckViolation:
		return apperrors.InvalidInput("validation_error", "folder data violates a constraint")
	default:
		return apperrors.Internal(apperrors.Code(fallbackCode), "folder storage operation failed")
	}
}

func folderNotFoundError() error {
	return apperrors.NotFound("not_found", "folder not found")
}
