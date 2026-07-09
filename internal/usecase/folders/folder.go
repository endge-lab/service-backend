package folders

import (
	"context"
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

const folderOperationTimeout = 15 * time.Second

var _ adapters.FolderService = (*Folder)(nil)

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

func (s *Folder) Create(
	ctx context.Context,
	input adapters.CreateFolderInput,
) (result *entities.Folder, err error) {
	const op = "folder.create"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}

	exists, err := s.folderRepository.ExistsByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.Conflict("identity_conflict", "folder identity already exists")
	}

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		return nil, err
	}

	folder := &entities.Folder{
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
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Folder) Update(
	ctx context.Context,
	input adapters.UpdateFolderInput,
) (result *entities.Folder, err error) {
	const op = "folder.update"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}

	current, err := s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
	if err != nil {
		return nil, err
	}

	parentID, err := s.resolveParentID(ctx, project.ID, input.EntityType, input.ParentIdentity)
	if err != nil {
		return nil, err
	}
	if current.IsRoot && !sameNullableUUID(current.ParentID, parentID) {
		return nil, apperrors.InvalidInput("validation_error", "root folder cannot be moved")
	}
	if err = s.validateNoCycle(ctx, current.ID, parentID); err != nil {
		return nil, err
	}

	updated := &entities.Folder{
		ID:          current.ID,
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
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Folder) GetByID(ctx context.Context, id uuid.UUID) (result *entities.Folder, err error) {
	const op = "folder.get_by_id"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if id == uuid.Nil {
		return nil, apperrors.InvalidInput("validation_error", "folder id is required")
	}

	return s.folderRepository.GetByID(ctx, id)
}

func (s *Folder) GetByIdentity(
	ctx context.Context,
	input adapters.GetFolderInput,
) (result *entities.Folder, err error) {
	const op = "folder.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.EntityType, &input.Identity); err != nil {
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}

	return s.folderRepository.GetByIdentity(ctx, &project.ID, input.EntityType, input.Identity)
}

func (s *Folder) List(
	ctx context.Context,
	input adapters.ListFoldersInput,
) (result []*entities.Folder, err error) {
	const op = "folder.list"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		return nil, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}

	return s.folderRepository.List(ctx, &project.ID, input.EntityType)
}

func (s *Folder) SoftDelete(ctx context.Context, input adapters.FolderIdentityInput) (err error) {
	const op = "folder.soft_delete"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, false)
	if err != nil {
		return err
	}

	return s.folderRepository.SoftDelete(ctx, folder.ID)
}

func (s *Folder) Restore(ctx context.Context, input adapters.FolderIdentityInput) (err error) {
	const op = "folder.restore"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		return err
	}

	return s.folderRepository.Restore(ctx, folder.ID)
}

func (s *Folder) HardDelete(ctx context.Context, input adapters.FolderIdentityInput) (err error) {
	const op = "folder.hard_delete"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	folder, err := s.resolveFolder(ctx, &input, true)
	if err != nil {
		return err
	}
	if folder.IsRoot && folder.IsSystem {
		return apperrors.Conflict(
			"system_folder_delete_forbidden",
			"system root folder cannot be hard deleted",
		)
	}

	return s.folderRepository.HardDelete(ctx, folder.ID)
}

func (s *Folder) Count(
	ctx context.Context,
	input adapters.ListFoldersInput,
) (result int64, err error) {
	const op = "folder.count"

	ctx, cancel := context.WithTimeout(ctx, folderOperationTimeout)
	defer cancel()

	ctx, observed := s.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input); err != nil {
		return 0, err
	}

	project, err := s.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return 0, err
	}

	return s.folderRepository.Count(ctx, &project.ID, input.EntityType)
}
