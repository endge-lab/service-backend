package folders

import (
	"context"
	"fmt"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/google/uuid"
)

type foldersRepositoryStub struct {
	folders           map[uuid.UUID]*entities.Folder
	foldersByIdentity map[string]*entities.Folder
	getByIdentityErr  error
}

func (s *foldersRepositoryStub) Create(
	context.Context,
	*entities.Folder,
) (*entities.Folder, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) Update(
	context.Context,
	*entities.Folder,
) (*entities.Folder, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.Folder, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) GetByIDIncludingDeleted(
	_ context.Context,
	id uuid.UUID,
) (*entities.Folder, error) {
	folder, ok := s.folders[id]
	if !ok {
		return nil, fmt.Errorf("folder %s not found", id)
	}

	return folder, nil
}

func (s *foldersRepositoryStub) GetByIdentity(
	_ context.Context,
	_ *uuid.UUID,
	_ entities.FolderEntityType,
	identity string,
) (*entities.Folder, error) {
	if s.getByIdentityErr != nil {
		return nil, s.getByIdentityErr
	}

	folder, ok := s.foldersByIdentity[identity]
	if !ok {
		return nil, apperrors.NotFound("not_found", "folder not found")
	}

	return folder, nil
}

func (s *foldersRepositoryStub) GetByIdentityIncludingDeleted(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (*entities.Folder, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) List(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
) ([]*entities.Folder, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) SoftDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *foldersRepositoryStub) Restore(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *foldersRepositoryStub) HardDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *foldersRepositoryStub) ExistsByIdentity(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (bool, error) {
	panic("not implemented")
}

func (s *foldersRepositoryStub) Count(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
) (int64, error) {
	panic("not implemented")
}

type projectsRepositoryStub struct {
	project *entities.Project
}

func (s *projectsRepositoryStub) Create(context.Context, *entities.Project) (*entities.Project, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.Project, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) GetByIdentity(_ context.Context, identity string) (*entities.Project, error) {
	if s.project == nil || s.project.Identity != identity {
		return nil, apperrors.NotFound("not_found", "project not found")
	}

	return s.project, nil
}

func (s *projectsRepositoryStub) GetByIdentityIncludingDeleted(context.Context, string) (*entities.Project, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) List(context.Context) ([]*entities.Project, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) Update(context.Context, *entities.Project) (*entities.Project, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) SoftDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectsRepositoryStub) Restore(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectsRepositoryStub) HardDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectsRepositoryStub) ExistsByIdentity(context.Context, string) (bool, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) Count(context.Context) (int64, error) {
	panic("not implemented")
}
