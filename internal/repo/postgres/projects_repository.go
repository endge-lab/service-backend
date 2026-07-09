package postgres

import (
	"context"
	stderrors "errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ProjectsRepository struct {
	*baseRepository
}

func NewProjectsRepository(
	queries *sqlc.Queries,
	tracer trace.Tracer,
	logger *zap.Logger,
) *ProjectsRepository {
	return &ProjectsRepository{
		baseRepository: newBaseRepository(queries, tracer, logger, "projects"),
	}
}

func (r *ProjectsRepository) Create(ctx context.Context, project *entities.Project) (result *entities.Project, err error) {
	const op = "repo.projects.Create"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	created, err := r.queries(ctx).CreateProject(ctx, mapCreateProjectParams(project))
	if err != nil {
		r.logger.Error("create project failed", zap.Error(err))
		return nil, mapProjectWriteError(err)
	}

	return mapProject(created), nil
}

func (r *ProjectsRepository) GetByID(ctx context.Context, id uuid.UUID) (result *entities.Project, err error) {
	const op = "repo.projects.GetByID"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	project, err := r.queries(ctx).GetProjectByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.logger.Error("get project by id failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mapProject(project), nil
}

func (r *ProjectsRepository) GetByIdentity(ctx context.Context, identity string) (result *entities.Project, err error) {
	const op = "repo.projects.GetByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	project, err := r.queries(ctx).GetProjectByIdentity(ctx, identity)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.logger.Error("get project by identity failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mapProject(project), nil
}

func (r *ProjectsRepository) GetByIdentityIncludingDeleted(
	ctx context.Context,
	identity string,
) (result *entities.Project, err error) {
	const op = "repo.projects.GetByIdentityIncludingDeleted"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	project, err := r.queries(ctx).GetProjectByIdentityIncludingDeleted(ctx, identity)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.logger.Error("get project by identity including deleted failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to get project")
	}

	return mapProject(project), nil
}

func (r *ProjectsRepository) List(ctx context.Context) (result []*entities.Project, err error) {
	const op = "repo.projects.List"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	projects, err := r.queries(ctx).ListProjects(ctx)
	if err != nil {
		r.logger.Error("list projects failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to list projects")
	}

	result = make([]*entities.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, mapProject(project))
	}

	return result, nil
}

func (r *ProjectsRepository) Update(ctx context.Context, project *entities.Project) (result *entities.Project, err error) {
	const op = "repo.projects.Update"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	updated, err := r.queries(ctx).UpdateProject(ctx, mapUpdateProjectParams(project))
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("not_found", "project not found")
		}

		r.logger.Error("update project failed", zap.Error(err))
		return nil, apperrors.Internal("internal_error", "failed to update project")
	}

	return mapProject(updated), nil
}

func (r *ProjectsRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.SoftDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := r.queries(ctx).SoftDeleteProject(ctx, id)
	if err != nil {
		r.logger.Error("soft delete project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to delete project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

func (r *ProjectsRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.Restore"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := r.queries(ctx).RestoreProject(ctx, id)
	if err != nil {
		r.logger.Error("restore project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to restore project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

func (r *ProjectsRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.HardDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	affected, err := r.queries(ctx).HardDeleteProject(ctx, id)
	if err != nil {
		r.logger.Error("hard delete project failed", zap.Error(err))
		return apperrors.Internal("internal_error", "failed to delete project")
	}
	if affected == 0 {
		return apperrors.NotFound("not_found", "project not found")
	}

	return nil
}

func (r *ProjectsRepository) ExistsByIdentity(ctx context.Context, identity string) (result bool, err error) {
	const op = "repo.projects.ExistsByIdentity"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	exists, err := r.queries(ctx).ExistsProjectByIdentity(ctx, identity)
	if err != nil {
		r.logger.Error("exists project by identity failed", zap.Error(err))
		return false, apperrors.Internal("internal_error", "failed to check project identity")
	}

	return exists, nil
}

func (r *ProjectsRepository) Count(ctx context.Context) (result int64, err error) {
	const op = "repo.projects.Count"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	count, err := r.queries(ctx).CountProject(ctx)
	if err != nil {
		r.logger.Error("count projects failed", zap.Error(err))
		return 0, apperrors.Internal("internal_error", "failed to count projects")
	}

	return count, nil
}

func mapProjectWriteError(err error) error {
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) &&
		pgErr.Code == postgresUniqueViolation &&
		pgErr.ConstraintName == "projects_identity_unique" {
		return apperrors.Conflict("identity_conflict", "project identity already exists")
	}

	return apperrors.Internal("internal_error", "failed to create project")
}
