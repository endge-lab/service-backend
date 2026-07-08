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
		return nil, apperrors.Internal("projects.create_failed", "failed to create project")
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
			return nil, apperrors.NotFound("projects.not_found", "project not found")
		}

		r.logger.Error("get project by id failed", zap.Error(err))
		return nil, apperrors.Internal("projects.get_by_id_failed", "failed to get project")
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
			return nil, apperrors.NotFound("projects.not_found", "project not found")
		}

		r.logger.Error("get project by identity failed", zap.Error(err))
		return nil, apperrors.Internal("projects.get_by_identity_failed", "failed to get project")
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
		return nil, apperrors.Internal("projects.list_failed", "failed to list projects")
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
			return nil, apperrors.NotFound("projects.not_found", "project not found")
		}

		r.logger.Error("update project failed", zap.Error(err))
		return nil, apperrors.Internal("projects.update_failed", "failed to update project")
	}

	return mapProject(updated), nil
}

func (r *ProjectsRepository) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.SoftDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	if err := r.queries(ctx).SoftDeleteProject(ctx, id); err != nil {
		r.logger.Error("soft delete project failed", zap.Error(err))
		return apperrors.Internal("projects.soft_delete_failed", "failed to delete project")
	}

	return nil
}

func (r *ProjectsRepository) Restore(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.Restore"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	if err := r.queries(ctx).RestoreProjects(ctx, id); err != nil {
		r.logger.Error("restore project failed", zap.Error(err))
		return apperrors.Internal("projects.restore_failed", "failed to restore project")
	}

	return nil
}

func (r *ProjectsRepository) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "repo.projects.HardDelete"

	ctx, step := telemetry.StartTrace(ctx, r.tracer, r.logger, op)
	defer func() { step.End(err) }()

	if err := r.queries(ctx).HardDeleteProject(ctx, id); err != nil {
		r.logger.Error("hard delete project failed", zap.Error(err))
		return apperrors.Internal("projects.hard_delete_failed", "failed to delete project")
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
		return false, apperrors.Internal("projects.exists_by_identity_failed", "failed to check project identity")
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
		return 0, apperrors.Internal("projects.count_failed", "failed to count projects")
	}

	return count, nil
}
