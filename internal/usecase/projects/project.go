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
	folderRepository  ports.FoldersRepository
	txManager         ports.TxManager
	observed          shared.ObservedUseCase
}

type ProjectParams struct {
	ProjectRepository ports.ProjectsRepository
	FolderRepository  ports.FoldersRepository
	TxManager         ports.TxManager
	Tracer            trace.Tracer
	Logger            *zap.Logger
	Metrics           *shared.UseCaseMetrics
}

func NewProjectService(params ProjectParams) *Project {
	return &Project{
		projectRepository: params.ProjectRepository,
		folderRepository:  params.FolderRepository,
		txManager:         params.TxManager,
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
		err = apperrors.Conflict("identity_conflict", "project identity already exists")
		observed.Logger().Error(op, zap.Error(err), zap.String("identity", input.Identity))
		return nil, err
	}

	project := projectFromCreateInput(input)

	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		result, err = s.projectRepository.Create(txCtx, project)
		if err != nil {
			return err
		}

		for _, root := range projectRootFolders(result.ID) {
			if _, err = s.folderRepository.Create(txCtx, root); err != nil {
				return err
			}
		}

		return nil
	})
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

func (s *Project) GetByIdentity(ctx context.Context, identity string) (result *entities.Project, err error) {
	const op = "project.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	identity = strings.TrimSpace(identity)

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
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

	return result, nil
}

func (s *Project) SoftDelete(ctx context.Context, identity string) (err error) {
	const op = "project.soft_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
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

	return nil
}

func (s *Project) Restore(ctx context.Context, identity string) (err error) {
	const op = "project.restore"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
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

	return nil
}

func (s *Project) HardDelete(ctx context.Context, identity string) (err error) {
	const op = "project.hard_delete"

	ctx, cancel := context.WithTimeout(ctx, projectOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
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
