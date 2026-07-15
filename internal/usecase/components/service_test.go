package components

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestComponentCreateAndUpdateReturnFolderIdentity(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-components"}
	repository := &componentRepositoryTestStub{
		createResult: &entities.Component{ID: uuid.New(), Identity: "card", FolderID: folder.ID},
		getResult:    &entities.Component{ID: uuid.New(), Identity: "card", ProjectID: project.ID, FolderID: folder.ID},
		updateResult: &entities.Component{ID: uuid.New(), Identity: "card", FolderID: folder.ID},
	}
	service := newComponentServiceForTest(project, &foldersRepositoryStub{folder: folder}, repository)

	created, err := service.Create(context.Background(), adapters.CreateComponentInput{
		ProjectIdentity: "demo", FolderIdentity: "root-components", Identity: "card", DisplayName: "Card",
		ComponentType: entities.ComponentTypeSFC, Source: "<template />",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.FolderIdentity != folder.Identity || repository.created.FolderID != folder.ID {
		t.Fatalf("create result = %#v, created = %#v", created, repository.created)
	}

	updated, err := service.Update(context.Background(), adapters.UpdateComponentInput{
		ProjectIdentity: "demo", ComponentIdentity: "card", FolderIdentity: "root-components", DisplayName: "Card",
		ComponentType: entities.ComponentTypeSFC, Source: "<template />",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.FolderIdentity != folder.Identity || repository.updated.FolderID != folder.ID {
		t.Fatalf("update result = %#v, updated = %#v", updated, repository.updated)
	}
}

func TestComponentCreateRejectsIdentityConflict(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-components"}
	service := newComponentServiceForTest(project, &foldersRepositoryStub{folder: folder}, &componentRepositoryTestStub{exists: true})

	_, err := service.Create(context.Background(), adapters.CreateComponentInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "card", DisplayName: "Card",
		ComponentType: entities.ComponentTypeSFC, Source: "<template />",
	})
	if got := apperrors.CodeOf(err); got != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", got)
	}
}

func TestComponentGetAndListResolveFoldersInUseCase(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	firstFolder := &entities.Folder{ID: uuid.New(), Identity: "root-components"}
	secondFolder := &entities.Folder{ID: uuid.New(), Identity: "forms"}
	firstComponent := &entities.Component{ID: uuid.New(), ProjectID: project.ID, FolderID: firstFolder.ID, Identity: "card"}
	secondComponent := &entities.Component{ID: uuid.New(), ProjectID: project.ID, FolderID: secondFolder.ID, Identity: "form"}
	folders := &foldersRepositoryStub{getByIDFolder: firstFolder, folders: []*entities.Folder{firstFolder, secondFolder}}
	repository := &componentRepositoryTestStub{getResult: firstComponent, listResult: []*entities.Component{firstComponent, secondComponent}}
	service := newComponentServiceForTest(project, folders, repository)

	got, err := service.GetByIdentity(context.Background(), adapters.GetComponentInput{ProjectIdentity: "demo", ComponentIdentity: "card"})
	if err != nil {
		t.Fatal(err)
	}
	if got.FolderIdentity != firstFolder.Identity || folders.getByIDCalls != 1 {
		t.Fatalf("get result = %#v, folder calls = %d", got, folders.getByIDCalls)
	}

	items, err := service.List(context.Background(), adapters.ListComponentsInput{ProjectIdentity: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].FolderIdentity != firstFolder.Identity || items[1].FolderIdentity != secondFolder.Identity {
		t.Fatalf("list result = %#v", items)
	}
	if folders.listCalls != 1 || folders.listEntityType != entities.FolderEntityTypeComponents || folders.getByIDCalls != 1 {
		t.Fatalf("list calls = %d, entity type = %q, get by ID calls = %d", folders.listCalls, folders.listEntityType, folders.getByIDCalls)
	}
}

func TestComponentListSkipsFoldersForEmptyResultAndDetectsUnavailableFolder(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folders := &foldersRepositoryStub{}
	service := newComponentServiceForTest(project, folders, &componentRepositoryTestStub{})

	items, err := service.List(context.Background(), adapters.ListComponentsInput{ProjectIdentity: "demo"})
	if err != nil || len(items) != 0 || folders.listCalls != 0 {
		t.Fatalf("items = %#v, err = %v, list calls = %d", items, err, folders.listCalls)
	}

	service = newComponentServiceForTest(project, folders, &componentRepositoryTestStub{listResult: []*entities.Component{{FolderID: uuid.New()}}})
	_, err = service.List(context.Background(), adapters.ListComponentsInput{ProjectIdentity: "demo"})
	if got := apperrors.CodeOf(err); got != "component_folder_not_found" {
		t.Fatalf("error code = %q, want component_folder_not_found", got)
	}
}

func TestComponentDeleteOperationsResolveExpectedRecord(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	active := &entities.Component{ID: uuid.New(), ProjectID: project.ID, Identity: "card"}
	deleted := &entities.Component{ID: uuid.New(), ProjectID: project.ID, Identity: "deleted-card"}
	repository := &componentRepositoryTestStub{getResult: active, getIncludingDeletedResult: deleted}
	service := newComponentServiceForTest(project, &foldersRepositoryStub{}, repository)

	if err := service.SoftDelete(context.Background(), adapters.ComponentIdentityInput{ProjectIdentity: "demo", ComponentIdentity: "card"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Restore(context.Background(), adapters.ComponentIdentityInput{ProjectIdentity: "demo", ComponentIdentity: "deleted-card"}); err != nil {
		t.Fatal(err)
	}
	if err := service.HardDelete(context.Background(), adapters.ComponentIdentityInput{ProjectIdentity: "demo", ComponentIdentity: "deleted-card"}); err != nil {
		t.Fatal(err)
	}
	if repository.softDeletedID != active.ID || repository.restoredID != deleted.ID || repository.hardDeletedID != deleted.ID {
		t.Fatalf("unexpected delete IDs: %#v", repository)
	}
}

func newComponentServiceForTest(
	project *entities.Project,
	folders *foldersRepositoryStub,
	components *componentRepositoryTestStub,
) *Component {
	return NewComponentService(ComponentParams{
		ProjectRepository:   &componentProjectRepositoryStub{project: project},
		FolderRepository:    folders,
		ComponentRepository: components,
		Tracer:              otel.Tracer("components-test"),
		Logger:              zap.NewNop(),
	})
}

type componentProjectRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.Project
	err     error
}

func (s *componentProjectRepositoryStub) GetByIdentity(context.Context, string) (*entities.Project, error) {
	return s.project, s.err
}

type componentRepositoryTestStub struct {
	ports.ComponentsRepository
	createResult              *entities.Component
	createErr                 error
	updateResult              *entities.Component
	updateErr                 error
	getResult                 *entities.Component
	getErr                    error
	getIncludingDeletedResult *entities.Component
	getIncludingDeletedErr    error
	listResult                []*entities.Component
	listErr                   error
	exists                    bool
	existsErr                 error
	created                   *entities.Component
	updated                   *entities.Component
	softDeletedID             uuid.UUID
	restoredID                uuid.UUID
	hardDeletedID             uuid.UUID
	softDeleteErr             error
	restoreErr                error
	hardDeleteErr             error
}

func (s *componentRepositoryTestStub) Create(_ context.Context, component *entities.Component) (*entities.Component, error) {
	s.created = component
	return s.createResult, s.createErr
}

func (s *componentRepositoryTestStub) Update(_ context.Context, component *entities.Component) (*entities.Component, error) {
	s.updated = component
	return s.updateResult, s.updateErr
}

func (s *componentRepositoryTestStub) GetByIdentity(context.Context, uuid.UUID, string) (*entities.Component, error) {
	return s.getResult, s.getErr
}

func (s *componentRepositoryTestStub) GetByIdentityIncludingDeleted(context.Context, uuid.UUID, string) (*entities.Component, error) {
	return s.getIncludingDeletedResult, s.getIncludingDeletedErr
}

func (s *componentRepositoryTestStub) List(context.Context, ports.ComponentsFilter) ([]*entities.Component, error) {
	return s.listResult, s.listErr
}

func (s *componentRepositoryTestStub) ExistsByIdentity(context.Context, uuid.UUID, string) (bool, error) {
	return s.exists, s.existsErr
}

func (s *componentRepositoryTestStub) SoftDelete(_ context.Context, id uuid.UUID) error {
	s.softDeletedID = id
	return s.softDeleteErr
}

func (s *componentRepositoryTestStub) Restore(_ context.Context, id uuid.UUID) error {
	s.restoredID = id
	return s.restoreErr
}

func (s *componentRepositoryTestStub) HardDelete(_ context.Context, id uuid.UUID) error {
	s.hardDeletedID = id
	return s.hardDeleteErr
}
