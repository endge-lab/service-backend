package projects

import (
	"context"
	"errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestCreateCreatesProjectAndRootFoldersInOneTransaction(t *testing.T) {
	projectID := uuid.New()
	projects := &projectsRepositoryStub{createdID: projectID}
	folders := &projectFoldersRepositoryStub{}
	tx := &txManagerStub{}
	service := NewProjectService(ProjectParams{
		ProjectRepository: projects,
		FolderRepository:  folders,
		TxManager:         tx,
		Tracer:            otel.Tracer("test"),
		Logger:            zap.NewNop(),
	})

	result, err := service.Create(context.Background(), CreateProjectInput{
		Identity:    "demo-project",
		DisplayName: "Demo Project",
		Active:      true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.ID != projectID {
		t.Fatalf("project ID = %s, want %s", result.ID, projectID)
	}
	if !tx.called || tx.rolledBack {
		t.Fatalf("transaction state: called=%v rolledBack=%v", tx.called, tx.rolledBack)
	}
	if len(folders.created) != 4 {
		t.Fatalf("created roots = %d, want 4", len(folders.created))
	}

	wantIdentities := []string{
		"root-components-legacy",
		"root-converters",
		"root-queries",
		"root-data-views",
	}
	for index, root := range folders.created {
		if root.Identity != wantIdentities[index] {
			t.Fatalf("root[%d] identity = %q, want %q", index, root.Identity, wantIdentities[index])
		}
		if root.ProjectID == nil || *root.ProjectID != projectID {
			t.Fatalf("root[%d] project ID = %v, want %s", index, root.ProjectID, projectID)
		}
		if !root.IsRoot || !root.IsSystem || root.ParentID != nil {
			t.Fatalf("root[%d] flags are invalid: %+v", index, root)
		}
	}
}

func TestCreateRollsBackWhenRootFolderCreationFails(t *testing.T) {
	expectedErr := errors.New("create root failed")
	tx := &txManagerStub{}
	service := NewProjectService(ProjectParams{
		ProjectRepository: &projectsRepositoryStub{createdID: uuid.New()},
		FolderRepository: &projectFoldersRepositoryStub{
			failAt: 2,
			err:    expectedErr,
		},
		TxManager: tx,
		Tracer:    otel.Tracer("test"),
		Logger:    zap.NewNop(),
	})

	_, err := service.Create(context.Background(), CreateProjectInput{
		Identity:    "demo-project",
		DisplayName: "Demo Project",
		Active:      true,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Create() error = %v, want %v", err, expectedErr)
	}
	if !tx.rolledBack {
		t.Fatal("transaction must be rolled back")
	}
}

type txManagerStub struct {
	called     bool
	rolledBack bool
}

func (s *txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	s.called = true
	if err := fn(ctx); err != nil {
		s.rolledBack = true
		return err
	}
	return nil
}

type projectsRepositoryStub struct {
	createdID uuid.UUID
}

func (s *projectsRepositoryStub) Create(
	_ context.Context,
	project *entities.RProject,
) (*entities.RProject, error) {
	created := *project
	created.ID = s.createdID
	return &created, nil
}

func (s *projectsRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.RProject, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) GetByIdentity(context.Context, string) (*entities.RProject, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) GetByIdentityIncludingDeleted(
	context.Context,
	string,
) (*entities.RProject, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) List(context.Context) ([]*entities.RProject, error) {
	panic("not implemented")
}

func (s *projectsRepositoryStub) Update(
	context.Context,
	*entities.RProject,
) (*entities.RProject, error) {
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
	return false, nil
}

func (s *projectsRepositoryStub) Count(context.Context) (int64, error) {
	panic("not implemented")
}

type projectFoldersRepositoryStub struct {
	created []*entities.RFolder
	failAt  int
	err     error
}

func (s *projectFoldersRepositoryStub) Create(
	_ context.Context,
	folder *entities.RFolder,
) (*entities.RFolder, error) {
	if s.err != nil && len(s.created) == s.failAt {
		return nil, s.err
	}
	s.created = append(s.created, folder)
	return folder, nil
}

func (s *projectFoldersRepositoryStub) Update(
	context.Context,
	*entities.RFolder,
) (*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) GetByIDIncludingDeleted(
	context.Context,
	uuid.UUID,
) (*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) GetByIdentity(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) GetByIdentityIncludingDeleted(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) List(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
) ([]*entities.RFolder, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) SoftDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) Restore(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) HardDelete(context.Context, uuid.UUID) error {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) ExistsByIdentity(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (bool, error) {
	panic("not implemented")
}

func (s *projectFoldersRepositoryStub) Count(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
) (int64, error) {
	panic("not implemented")
}
