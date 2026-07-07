package projects

import (
	"context"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const projectOperationTimeout = 15 * time.Second

var _ adapters.ProjectService = (*Project)(nil)

type Project struct {
	projectRepository ports.ProjectsRepository
	observed          shared.ObservedUseCase
}

type ProjectParams struct {
	ProjectRepository ports.ProjectsRepository
	Tracer            trace.Tracer
	Logger            *zap.Logger
	Metrics           *shared.UseCaseMetrics
}

func NewProjectService(params ProjectParams) *Project {
	return &Project{
		projectRepository: params.ProjectRepository,
		observed: shared.NewObservedUseCase(
			params.Tracer,
			params.Logger,
			params.Metrics,
		),
	}
}

func (s *Project) Create(ctx context.Context, input adapters.CreateProjectInput) (result *entities.Project, err error) {
	const op = "project.create"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err := normalizeAndValidateCreateProjectInput(&input); err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	exists, err := s.projectRepository.ExistsByIdentity(ctx, input.Identity)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	if exists {
		err = apperrors.Conflict("projects.identity_already_exists", "project identity already exists")
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
		return nil, err
	}

	project := projectFromCreateInput(input)

	result, err = s.projectRepository.Create(ctx, project)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Project) GetByID(ctx context.Context, id uuid.UUID) (result *entities.Project, err error) {
	const op = "project.get_by_id"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("projects.empty_id", "project id is required")
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

func (s *Project) GetByIdentity(ctx context.Context, identity string) (result *entities.Project, err error) {
	const op = "project.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	identity = strings.TrimSpace(identity)

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if identity == "" {
		err = apperrors.InvalidInput("projects.empty_identity", "project identity is required")
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

func (s *Project) List(ctx context.Context) (result []*entities.Project, err error) {
	const op = "project.list"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	result, err = s.projectRepository.List(ctx)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Project) Update(ctx context.Context, input adapters.UpdateProjectInput) (result *entities.Project, err error) {
	const op = "project.update"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err := normalizeAndValidateUpdateProjectInput(&input); err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	current, err := s.projectRepository.GetByID(ctx, input.ID)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("project_id", input.ID.String()))
		return nil, err
	}

	if current.Identity != input.Identity {
		exists, err := s.projectRepository.ExistsByIdentity(ctx, input.Identity)
		if err != nil {
			observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
			return nil, err
		}

		if exists {
			err = apperrors.Conflict("projects.identity_already_exists", "project identity already exists")
			observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
			return nil, err
		}
	}

	project := projectFromUpdateInput(input)

	result, err = s.projectRepository.Update(ctx, project)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Project) SoftDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "project.soft_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("projects.empty_id", "project id is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	err = s.projectRepository.SoftDelete(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	return nil
}

func (s *Project) Restore(ctx context.Context, id uuid.UUID) (err error) {
	const op = "project.restore"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("projects.empty_id", "project id is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	err = s.projectRepository.Restore(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	return nil
}

func (s *Project) HardDelete(ctx context.Context, id uuid.UUID) (err error) {
	const op = "project.hard_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		err = apperrors.InvalidInput("projects.empty_id", "project id is required")
		observed.Logger().Error(op, zap.Error(err))
		return err
	}

	err = s.projectRepository.HardDelete(ctx, id)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	return nil
}

func (s *Project) Count(ctx context.Context) (result int64, err error) {
	const op = "project.count"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	result, err = s.projectRepository.Count(ctx)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return 0, err
	}

	return result, nil
}

func normalizeAndValidateCreateProjectInput(input *adapters.CreateProjectInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Identity == "" {
		return apperrors.InvalidInput("projects.empty_identity", "project identity is required")
	}

	if input.DisplayName == "" {
		return apperrors.InvalidInput("projects.empty_display_name", "project display name is required")
	}

	if input.Meta == nil {
		input.Meta = map[string]any{}
	}

	if input.AllowedEnvironmentIDs == nil {
		input.AllowedEnvironmentIDs = []uuid.UUID{}
	}

	return nil
}
