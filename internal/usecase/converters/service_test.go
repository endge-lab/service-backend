package converters

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

func TestConverterCreateAndUpdateReturnFolderIdentity(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-converters"}
	repository := &converterRepositoryTestStub{
		createResult: &entities.Converter{ID: uuid.New(), Identity: "date-format", FolderID: folder.ID},
		getResult:    &entities.Converter{ID: uuid.New(), Identity: "date-format", ProjectID: project.ID, FolderID: folder.ID},
		updateResult: &entities.Converter{ID: uuid.New(), Identity: "date-format", FolderID: folder.ID},
	}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{folder: folder}, repository)

	created, err := service.Create(context.Background(), adapters.CreateConverterInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "date-format", DisplayName: "Date format", ConverterType: "format",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.FolderIdentity != folder.Identity || repository.created.FolderID != folder.ID {
		t.Fatalf("create result = %#v, created = %#v", created, repository.created)
	}

	updated, err := service.Update(context.Background(), adapters.UpdateConverterInput{
		ProjectIdentity: "demo", ConverterIdentity: "date-format", FolderIdentity: folder.Identity, DisplayName: "Date format", ConverterType: "format",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.FolderIdentity != folder.Identity || repository.updated.FolderID != folder.ID {
		t.Fatalf("update result = %#v, updated = %#v", updated, repository.updated)
	}
}

func TestConverterCreateRejectsIdentityConflict(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-converters"}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{folder: folder}, &converterRepositoryTestStub{exists: true})

	_, err := service.Create(context.Background(), adapters.CreateConverterInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "date-format", DisplayName: "Date format", ConverterType: "format",
	})
	if got := apperrors.CodeOf(err); got != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", got)
	}
}

func TestConverterGetAndListResolveFoldersInUseCase(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	firstFolder := &entities.Folder{ID: uuid.New(), Identity: "root-converters"}
	secondFolder := &entities.Folder{ID: uuid.New(), Identity: "formatters"}
	firstConverter := &entities.Converter{ID: uuid.New(), ProjectID: project.ID, FolderID: firstFolder.ID, Identity: "date-format"}
	secondConverter := &entities.Converter{ID: uuid.New(), ProjectID: project.ID, FolderID: secondFolder.ID, Identity: "currency-format"}
	folders := &foldersRepositoryStub{getByIDFolder: firstFolder, folders: []*entities.Folder{firstFolder, secondFolder}}
	repository := &converterRepositoryTestStub{getResult: firstConverter, listResult: []*entities.Converter{firstConverter, secondConverter}}
	service := newConverterServiceForTest(project, folders, repository)

	got, err := service.GetByIdentity(context.Background(), adapters.GetConverterInput{ProjectIdentity: "demo", ConverterIdentity: "date-format"})
	if err != nil {
		t.Fatal(err)
	}
	if got.FolderIdentity != firstFolder.Identity || folders.getByIDCalls != 1 {
		t.Fatalf("get result = %#v, folder calls = %d", got, folders.getByIDCalls)
	}

	items, err := service.List(context.Background(), adapters.ListConvertersInput{ProjectIdentity: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].FolderIdentity != firstFolder.Identity || items[1].FolderIdentity != secondFolder.Identity {
		t.Fatalf("list result = %#v", items)
	}
	if folders.listCalls != 1 || folders.listEntityType != entities.FolderEntityTypeConverters || folders.getByIDCalls != 1 {
		t.Fatalf("list calls = %d, entity type = %q, get by ID calls = %d", folders.listCalls, folders.listEntityType, folders.getByIDCalls)
	}
}

func TestConverterListSkipsFoldersForEmptyResultAndDetectsUnavailableFolder(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folders := &foldersRepositoryStub{}
	service := newConverterServiceForTest(project, folders, &converterRepositoryTestStub{})

	items, err := service.List(context.Background(), adapters.ListConvertersInput{ProjectIdentity: "demo"})
	if err != nil || len(items) != 0 || folders.listCalls != 0 {
		t.Fatalf("items = %#v, err = %v, list calls = %d", items, err, folders.listCalls)
	}

	service = newConverterServiceForTest(project, folders, &converterRepositoryTestStub{listResult: []*entities.Converter{{FolderID: uuid.New()}}})
	_, err = service.List(context.Background(), adapters.ListConvertersInput{ProjectIdentity: "demo"})
	if got := apperrors.CodeOf(err); got != "converter_folder_not_found" {
		t.Fatalf("error code = %q, want converter_folder_not_found", got)
	}
}

func TestConverterDeleteOperations(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	active := &entities.Converter{ID: uuid.New(), ProjectID: project.ID, Identity: "date-format"}
	deleted := &entities.Converter{ID: uuid.New(), ProjectID: project.ID, Identity: "deleted-format"}
	repository := &converterRepositoryTestStub{getResult: active, getIncludingDeletedResult: deleted}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{}, repository)

	if err := service.SoftDelete(context.Background(), adapters.ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "date-format"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Restore(context.Background(), adapters.ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "deleted-format"}); err != nil {
		t.Fatal(err)
	}
	if err := service.HardDelete(context.Background(), adapters.ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "deleted-format"}); err != nil {
		t.Fatal(err)
	}
	if repository.softDeletedID != active.ID || repository.restoredID != deleted.ID || repository.hardDeletedID != deleted.ID {
		t.Fatalf("unexpected delete IDs: %#v", repository)
	}
}

func TestConverterHardDeleteRejectsSystemConverter(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	systemConverter := &entities.Converter{ID: uuid.New(), ProjectID: project.ID, Identity: "system", IsSystem: true}
	repository := &converterRepositoryTestStub{getIncludingDeletedResult: systemConverter}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{}, repository)

	err := service.HardDelete(context.Background(), adapters.ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "system"})
	if got := apperrors.CodeOf(err); got != "system_converter_delete_forbidden" {
		t.Fatalf("error code = %q, want system_converter_delete_forbidden", got)
	}
	if repository.hardDeletedID != uuid.Nil {
		t.Fatalf("hard delete called for system converter: %s", repository.hardDeletedID)
	}
}

func newConverterServiceForTest(
	project *entities.Project,
	folders *foldersRepositoryStub,
	converters *converterRepositoryTestStub,
) *Converter {
	return NewConverterService(ConverterParams{
		ProjectRepository:   &converterProjectRepositoryStub{project: project},
		FolderRepository:    folders,
		ConverterRepository: converters,
		Tracer:              otel.Tracer("converters-test"),
		Logger:              zap.NewNop(),
	})
}

type converterProjectRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.Project
	err     error
}

func (s *converterProjectRepositoryStub) GetByIdentity(context.Context, string) (*entities.Project, error) {
	return s.project, s.err
}

type converterRepositoryTestStub struct {
	ports.ConvertersRepository
	createResult              *entities.Converter
	createErr                 error
	updateResult              *entities.Converter
	updateErr                 error
	getResult                 *entities.Converter
	getErr                    error
	getIncludingDeletedResult *entities.Converter
	getIncludingDeletedErr    error
	listResult                []*entities.Converter
	listErr                   error
	exists                    bool
	existsErr                 error
	created                   *entities.Converter
	updated                   *entities.Converter
	softDeletedID             uuid.UUID
	restoredID                uuid.UUID
	hardDeletedID             uuid.UUID
	softDeleteErr             error
	restoreErr                error
	hardDeleteErr             error
}

func (s *converterRepositoryTestStub) Create(_ context.Context, converter *entities.Converter) (*entities.Converter, error) {
	s.created = converter
	return s.createResult, s.createErr
}

func (s *converterRepositoryTestStub) Update(_ context.Context, converter *entities.Converter) (*entities.Converter, error) {
	s.updated = converter
	return s.updateResult, s.updateErr
}

func (s *converterRepositoryTestStub) GetByIdentity(context.Context, uuid.UUID, string) (*entities.Converter, error) {
	return s.getResult, s.getErr
}

func (s *converterRepositoryTestStub) GetByIdentityIncludingDeleted(context.Context, uuid.UUID, string) (*entities.Converter, error) {
	return s.getIncludingDeletedResult, s.getIncludingDeletedErr
}

func (s *converterRepositoryTestStub) List(context.Context, ports.ConvertersFilter) ([]*entities.Converter, error) {
	return s.listResult, s.listErr
}

func (s *converterRepositoryTestStub) ExistsByIdentity(context.Context, uuid.UUID, string) (bool, error) {
	return s.exists, s.existsErr
}

func (s *converterRepositoryTestStub) SoftDelete(_ context.Context, id uuid.UUID) error {
	s.softDeletedID = id
	return s.softDeleteErr
}

func (s *converterRepositoryTestStub) Restore(_ context.Context, id uuid.UUID) error {
	s.restoredID = id
	return s.restoreErr
}

func (s *converterRepositoryTestStub) HardDelete(_ context.Context, id uuid.UUID) error {
	s.hardDeletedID = id
	return s.hardDeleteErr
}
