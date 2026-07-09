package folders

import (
	"context"
	"fmt"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type foldersRepositoryStub struct {
	folders map[uuid.UUID]*entities.Folder
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
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (*entities.Folder, error) {
	panic("not implemented")
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
