package workspaces

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestCreateProvisionsTenantRootFolderInTransaction(t *testing.T) {
	repository := &repositoryStub{}
	folders := &workspaceFoldersStub{}
	tx := &workspaceTxStub{}
	service := NewWorkspaceService(WorkspaceParams{
		Repository: repository, FolderRepository: folders, TxManager: tx,
		Observability: observability.NewCore(otel.Tracer("test"), zap.NewNop()),
	})

	workspace, err := service.Create(context.Background(), CreateWorkspaceInput{Identity: "default", DisplayName: "Default"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if !tx.called {
		t.Fatal("workspace creation must use a transaction when root provisioning is configured")
	}
	if len(folders.created) != 1 {
		t.Fatalf("created root count = %d, want 1", len(folders.created))
	}
	root := folders.created[0]
	if root.WorkspaceID != workspace.ID || root.ProjectID != nil || root.EntityType != entities.FolderEntityTypeTenants || root.Identity != entities.TenantRootFolderIdentity || !root.IsRoot || !root.IsSystem {
		t.Fatalf("unexpected tenant root: %+v", root)
	}
}

type workspaceTxStub struct{ called bool }

func (s *workspaceTxStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	s.called = true
	return fn(ctx)
}

type workspaceFoldersStub struct{ created []*entities.RFolder }

var _ ports.FoldersRepository = (*workspaceFoldersStub)(nil)

func (s *workspaceFoldersStub) Create(_ context.Context, folder *entities.RFolder) (*entities.RFolder, error) {
	copy := *folder
	copy.ID = uuid.New()
	s.created = append(s.created, &copy)
	return &copy, nil
}
func (s *workspaceFoldersStub) Update(context.Context, *entities.RFolder) (*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) GetByID(context.Context, uuid.UUID) (*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) GetByIDIncludingDeleted(context.Context, uuid.UUID) (*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) GetByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) GetByIdentityIncludingDeleted(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) List(context.Context, *uuid.UUID, entities.FolderEntityType) ([]*entities.RFolder, error) {
	return nil, nil
}
func (s *workspaceFoldersStub) SoftDelete(context.Context, uuid.UUID) error { return nil }
func (s *workspaceFoldersStub) Restore(context.Context, uuid.UUID) error    { return nil }
func (s *workspaceFoldersStub) HardDelete(context.Context, uuid.UUID) error { return nil }
func (s *workspaceFoldersStub) ExistsByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (bool, error) {
	return false, nil
}
func (s *workspaceFoldersStub) Count(context.Context, *uuid.UUID, entities.FolderEntityType) (int64, error) {
	return 0, nil
}
