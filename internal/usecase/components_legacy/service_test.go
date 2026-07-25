package components_legacy

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestComponentLegacyCreateAndUpdateReturnFolderIdentity(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-components-legacy"}
	repository := &componentRepositoryTestStub{
		createResult: &entities.RComponentLegacy{ID: uuid.New(), Identity: "card", FolderID: folder.ID},
		getResult:    &entities.RComponentLegacy{ID: uuid.New(), Identity: "card", ProjectID: project.ID, FolderID: folder.ID},
		updateResult: &entities.RComponentLegacy{ID: uuid.New(), Identity: "card", FolderID: folder.ID},
	}
	service := newComponentLegacyServiceForTest(project, &foldersRepositoryStub{folder: folder}, repository)

	created, err := service.Create(context.Background(), CreateComponentLegacyInput{
		ProjectIdentity: "demo", FolderIdentity: "root-components-legacy", Identity: "card", DisplayName: "Card",
		ComponentType: entities.RComponentLegacyTypeSFC, Source: "<template />",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.FolderIdentity != folder.Identity || repository.created.FolderID != folder.ID {
		t.Fatalf("create result = %#v, created = %#v", created, repository.created)
	}

	updated, err := service.Update(context.Background(), UpdateComponentLegacyInput{
		ProjectIdentity: "demo", ComponentLegacyIdentity: "card", FolderIdentity: "root-components-legacy", DisplayName: "Card",
		ComponentType: entities.RComponentLegacyTypeSFC, Source: "<template />",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.FolderIdentity != folder.Identity || repository.updated.FolderID != folder.ID {
		t.Fatalf("update result = %#v, updated = %#v", updated, repository.updated)
	}
}

func TestComponentLegacyCreateRejectsIdentityConflict(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-components-legacy"}
	service := newComponentLegacyServiceForTest(project, &foldersRepositoryStub{folder: folder}, &componentRepositoryTestStub{exists: true})

	_, err := service.Create(context.Background(), CreateComponentLegacyInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "card", DisplayName: "Card",
		ComponentType: entities.RComponentLegacyTypeSFC, Source: "<template />",
	})
	if got := apperrors.CodeOf(err); got != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", got)
	}
}

func TestComponentLegacyGetAndListResolveFoldersInUseCase(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	firstFolder := &entities.RFolder{ID: uuid.New(), Identity: "root-components-legacy"}
	secondFolder := &entities.RFolder{ID: uuid.New(), Identity: "forms"}
	firstComponentLegacy := &entities.RComponentLegacy{ID: uuid.New(), ProjectID: project.ID, FolderID: firstFolder.ID, Identity: "card"}
	secondComponentLegacy := &entities.RComponentLegacy{ID: uuid.New(), ProjectID: project.ID, FolderID: secondFolder.ID, Identity: "form"}
	folders := &foldersRepositoryStub{getByIDFolder: firstFolder, folders: []*entities.RFolder{firstFolder, secondFolder}}
	repository := &componentRepositoryTestStub{getResult: firstComponentLegacy, listResult: []*entities.RComponentLegacy{firstComponentLegacy, secondComponentLegacy}}
	service := newComponentLegacyServiceForTest(project, folders, repository)

	got, err := service.GetByIdentity(context.Background(), GetComponentLegacyInput{ProjectIdentity: "demo", ComponentLegacyIdentity: "card"})
	if err != nil {
		t.Fatal(err)
	}
	if got.FolderIdentity != firstFolder.Identity || folders.getByIDCalls != 1 {
		t.Fatalf("get result = %#v, folder calls = %d", got, folders.getByIDCalls)
	}

	items, err := service.List(context.Background(), ListComponentsLegacyInput{ProjectIdentity: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].FolderIdentity != firstFolder.Identity || items[1].FolderIdentity != secondFolder.Identity {
		t.Fatalf("list result = %#v", items)
	}
	if folders.listCalls != 1 || folders.listEntityType != entities.FolderEntityTypeComponentsLegacy || folders.getByIDCalls != 1 {
		t.Fatalf("list calls = %d, entity type = %q, get by ID calls = %d", folders.listCalls, folders.listEntityType, folders.getByIDCalls)
	}
}

func TestComponentLegacyListSkipsFoldersForEmptyResultAndDetectsUnavailableFolder(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folders := &foldersRepositoryStub{}
	service := newComponentLegacyServiceForTest(project, folders, &componentRepositoryTestStub{})

	items, err := service.List(context.Background(), ListComponentsLegacyInput{ProjectIdentity: "demo"})
	if err != nil || len(items) != 0 || folders.listCalls != 0 {
		t.Fatalf("items = %#v, err = %v, list calls = %d", items, err, folders.listCalls)
	}

	service = newComponentLegacyServiceForTest(project, folders, &componentRepositoryTestStub{listResult: []*entities.RComponentLegacy{{FolderID: uuid.New()}}})
	_, err = service.List(context.Background(), ListComponentsLegacyInput{ProjectIdentity: "demo"})
	if got := apperrors.CodeOf(err); got != "component_folder_not_found" {
		t.Fatalf("error code = %q, want component_folder_not_found", got)
	}
}

func TestComponentLegacyDeleteOperationsResolveExpectedRecord(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	active := &entities.RComponentLegacy{ID: uuid.New(), ProjectID: project.ID, Identity: "card"}
	deleted := &entities.RComponentLegacy{ID: uuid.New(), ProjectID: project.ID, Identity: "deleted-card"}
	repository := &componentRepositoryTestStub{getResult: active, getIncludingDeletedResult: deleted}
	service := newComponentLegacyServiceForTest(project, &foldersRepositoryStub{}, repository)

	if err := service.SoftDelete(context.Background(), ComponentLegacyIdentityInput{ProjectIdentity: "demo", ComponentLegacyIdentity: "card"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Restore(context.Background(), ComponentLegacyIdentityInput{ProjectIdentity: "demo", ComponentLegacyIdentity: "deleted-card"}); err != nil {
		t.Fatal(err)
	}
	if err := service.HardDelete(context.Background(), ComponentLegacyIdentityInput{ProjectIdentity: "demo", ComponentLegacyIdentity: "deleted-card"}); err != nil {
		t.Fatal(err)
	}
	if repository.softDeletedID != active.ID || repository.restoredID != deleted.ID || repository.hardDeletedID != deleted.ID {
		t.Fatalf("unexpected delete IDs: %#v", repository)
	}
}

func newComponentLegacyServiceForTest(
	project *entities.RProject,
	folders *foldersRepositoryStub,
	components *componentRepositoryTestStub,
) *ComponentLegacy {
	return NewComponentLegacyService(ComponentLegacyParams{
		ProjectRepository:         &componentProjectRepositoryStub{project: project},
		FolderRepository:          folders,
		ComponentLegacyRepository: components,
		Observability:             observability.NewCore(otel.Tracer("components-test"), zap.NewNop()),
	})
}

type componentProjectRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.RProject
	err     error
}

func (s *componentProjectRepositoryStub) GetByIdentity(context.Context, string) (*entities.RProject, error) {
	return s.project, s.err
}

type componentRepositoryTestStub struct {
	ports.ComponentsLegacyRepository
	createResult              *entities.RComponentLegacy
	createErr                 error
	updateResult              *entities.RComponentLegacy
	updateErr                 error
	getResult                 *entities.RComponentLegacy
	getErr                    error
	getIncludingDeletedResult *entities.RComponentLegacy
	getIncludingDeletedErr    error
	listResult                []*entities.RComponentLegacy
	listErr                   error
	exists                    bool
	existsErr                 error
	created                   *entities.RComponentLegacy
	updated                   *entities.RComponentLegacy
	softDeletedID             uuid.UUID
	restoredID                uuid.UUID
	hardDeletedID             uuid.UUID
	softDeleteErr             error
	restoreErr                error
	hardDeleteErr             error
}

func (s *componentRepositoryTestStub) Create(_ context.Context, component *entities.RComponentLegacy) (*entities.RComponentLegacy, error) {
	s.created = component
	return s.createResult, s.createErr
}

func (s *componentRepositoryTestStub) Update(_ context.Context, component *entities.RComponentLegacy) (*entities.RComponentLegacy, error) {
	s.updated = component
	return s.updateResult, s.updateErr
}

func (s *componentRepositoryTestStub) GetByIdentity(context.Context, uuid.UUID, string) (*entities.RComponentLegacy, error) {
	return s.getResult, s.getErr
}

func (s *componentRepositoryTestStub) GetByIdentityIncludingDeleted(context.Context, uuid.UUID, string) (*entities.RComponentLegacy, error) {
	return s.getIncludingDeletedResult, s.getIncludingDeletedErr
}

func (s *componentRepositoryTestStub) List(context.Context, ports.ComponentsLegacyFilter) ([]*entities.RComponentLegacy, error) {
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
