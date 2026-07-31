package converters

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestConverterCreateAndUpdateReturnFolderIdentity(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-converters"}
	repository := &converterRepositoryTestStub{
		createResult: &entities.RConverter{ID: uuid.New(), Identity: "date-format", FolderID: folder.ID},
		getResult:    &entities.RConverter{ID: uuid.New(), Identity: "date-format", ProjectID: project.ID, FolderID: folder.ID},
		updateResult: &entities.RConverter{ID: uuid.New(), Identity: "date-format", FolderID: folder.ID},
	}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{folder: folder}, repository)

	created, err := service.Create(converterContext(project), CreateConverterInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "date-format", DisplayName: "Date format", ConverterType: "format",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.FolderIdentity != folder.Identity || repository.created.FolderID != folder.ID {
		t.Fatalf("create result = %#v, created = %#v", created, repository.created)
	}

	updated, err := service.Update(converterContext(project), UpdateConverterInput{
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
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-converters"}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{folder: folder}, &converterRepositoryTestStub{exists: true})

	_, err := service.Create(converterContext(project), CreateConverterInput{
		ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "date-format", DisplayName: "Date format", ConverterType: "format",
	})
	if got := apperrors.CodeOf(err); got != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", got)
	}
}

func TestConverterCreateRecordsBusinessSteps(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-converters"}
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	defer func() { _ = provider.Shutdown(context.Background()) }()
	logCore, logs := observer.New(zap.InfoLevel)
	service := NewConverterService(ConverterParams{
		ProjectRepository: &converterProjectRepositoryStub{project: project},
		FolderRepository:  &foldersRepositoryStub{folder: folder},
		ConverterRepository: &converterRepositoryTestStub{
			createResult: &entities.RConverter{ID: uuid.New(), Identity: "date-format", FolderID: folder.ID},
		},
		Observability: observability.NewCore(provider.Tracer("converter-test"), zap.New(logCore)),
	})

	_, err := service.Create(converterContext(project), CreateConverterInput{
		ProjectIdentity: "demo",
		FolderIdentity:  folder.Identity,
		Identity:        "date-format",
		DisplayName:     "Date format",
		ConverterType:   "format",
	})
	if err != nil {
		t.Fatal(err)
	}

	ended := spanRecorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	wantEvents := []string{
		"converter.create.input_validated",
		"converter.create.project_resolved",
		"converter.create.folder_resolved",
		"converter.create.identity_available",
		"converter.create.persisted",
		"converter.create.completed",
	}
	events := ended[0].Events()
	if len(events) != len(wantEvents) {
		t.Fatalf("events = %#v, want %v", events, wantEvents)
	}
	for index, want := range wantEvents {
		if events[index].Name != want {
			t.Fatalf("event[%d] = %q, want %q", index, events[index].Name, want)
		}
	}
	if len(logs.FilterMessage("converter persisted").All()) != 1 {
		t.Fatalf("persisted log entries = %#v, want one", logs.All())
	}
}

func TestConverterGetAndListResolveFoldersInUseCase(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	firstFolder := &entities.RFolder{ID: uuid.New(), Identity: "root-converters"}
	secondFolder := &entities.RFolder{ID: uuid.New(), Identity: "formatters"}
	firstConverter := &entities.RConverter{ID: uuid.New(), ProjectID: project.ID, FolderID: firstFolder.ID, Identity: "date-format"}
	secondConverter := &entities.RConverter{ID: uuid.New(), ProjectID: project.ID, FolderID: secondFolder.ID, Identity: "currency-format"}
	folders := &foldersRepositoryStub{getByIDFolder: firstFolder, folders: []*entities.RFolder{firstFolder, secondFolder}}
	repository := &converterRepositoryTestStub{getResult: firstConverter, listResult: []*entities.RConverter{firstConverter, secondConverter}}
	service := newConverterServiceForTest(project, folders, repository)

	got, err := service.GetByIdentity(converterContext(project), GetConverterInput{ProjectIdentity: "demo", ConverterIdentity: "date-format"})
	if err != nil {
		t.Fatal(err)
	}
	if got.FolderIdentity != firstFolder.Identity || folders.getByIDCalls != 1 {
		t.Fatalf("get result = %#v, folder calls = %d", got, folders.getByIDCalls)
	}

	items, err := service.List(converterContext(project), ListConvertersInput{ProjectIdentity: "demo"})
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
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folders := &foldersRepositoryStub{}
	service := newConverterServiceForTest(project, folders, &converterRepositoryTestStub{})

	items, err := service.List(converterContext(project), ListConvertersInput{ProjectIdentity: "demo"})
	if err != nil || len(items) != 0 || folders.listCalls != 0 {
		t.Fatalf("items = %#v, err = %v, list calls = %d", items, err, folders.listCalls)
	}

	service = newConverterServiceForTest(project, folders, &converterRepositoryTestStub{listResult: []*entities.RConverter{{FolderID: uuid.New()}}})
	_, err = service.List(converterContext(project), ListConvertersInput{ProjectIdentity: "demo"})
	if got := apperrors.CodeOf(err); got != "converter_folder_not_found" {
		t.Fatalf("error code = %q, want converter_folder_not_found", got)
	}
}

func TestConverterDeleteOperations(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	active := &entities.RConverter{ID: uuid.New(), ProjectID: project.ID, Identity: "date-format"}
	deleted := &entities.RConverter{ID: uuid.New(), ProjectID: project.ID, Identity: "deleted-format"}
	repository := &converterRepositoryTestStub{getResult: active, getIncludingDeletedResult: deleted}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{}, repository)

	if err := service.SoftDelete(converterContext(project), ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "date-format"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Restore(converterContext(project), ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "deleted-format"}); err != nil {
		t.Fatal(err)
	}
	if err := service.HardDelete(converterContext(project), ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "deleted-format"}); err != nil {
		t.Fatal(err)
	}
	if repository.softDeletedID != active.ID || repository.restoredID != deleted.ID || repository.hardDeletedID != deleted.ID {
		t.Fatalf("unexpected delete IDs: %#v", repository)
	}
}

func TestConverterHardDeleteRejectsSystemConverter(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	systemConverter := &entities.RConverter{ID: uuid.New(), ProjectID: project.ID, Identity: "system", IsSystem: true}
	repository := &converterRepositoryTestStub{getIncludingDeletedResult: systemConverter}
	service := newConverterServiceForTest(project, &foldersRepositoryStub{}, repository)

	err := service.HardDelete(converterContext(project), ConverterIdentityInput{ProjectIdentity: "demo", ConverterIdentity: "system"})
	if got := apperrors.CodeOf(err); got != "system_converter_delete_forbidden" {
		t.Fatalf("error code = %q, want system_converter_delete_forbidden", got)
	}
	if repository.hardDeletedID != uuid.Nil {
		t.Fatalf("hard delete called for system converter: %s", repository.hardDeletedID)
	}
}

func newConverterServiceForTest(
	project *entities.RProject,
	folders *foldersRepositoryStub,
	converters *converterRepositoryTestStub,
) *Converter {
	if project.WorkspaceID == uuid.Nil {
		project.WorkspaceID = uuid.New()
	}
	return NewConverterService(ConverterParams{
		ProjectRepository:   &converterProjectRepositoryStub{project: project},
		FolderRepository:    folders,
		ConverterRepository: converters,
		Observability:       observability.NewCore(otel.Tracer("converters-test"), zap.NewNop()),
	})
}

func converterContext(project *entities.RProject) context.Context {
	if project.WorkspaceID == uuid.Nil {
		project.WorkspaceID = uuid.New()
	}
	return entities.WithWorkspaceID(context.Background(), project.WorkspaceID)
}

type converterProjectRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.RProject
	err     error
}

func (s *converterProjectRepositoryStub) GetByIdentity(context.Context, string) (*entities.RProject, error) {
	return s.project, s.err
}

type converterRepositoryTestStub struct {
	ports.ConvertersRepository
	createResult              *entities.RConverter
	createErr                 error
	updateResult              *entities.RConverter
	updateErr                 error
	getResult                 *entities.RConverter
	getErr                    error
	getIncludingDeletedResult *entities.RConverter
	getIncludingDeletedErr    error
	listResult                []*entities.RConverter
	listErr                   error
	exists                    bool
	existsErr                 error
	created                   *entities.RConverter
	updated                   *entities.RConverter
	softDeletedID             uuid.UUID
	restoredID                uuid.UUID
	hardDeletedID             uuid.UUID
	softDeleteErr             error
	restoreErr                error
	hardDeleteErr             error
}

func (s *converterRepositoryTestStub) Create(_ context.Context, converter *entities.RConverter) (*entities.RConverter, error) {
	s.created = converter
	return s.createResult, s.createErr
}

func (s *converterRepositoryTestStub) Update(_ context.Context, converter *entities.RConverter) (*entities.RConverter, error) {
	s.updated = converter
	return s.updateResult, s.updateErr
}

func (s *converterRepositoryTestStub) GetByIdentity(context.Context, uuid.UUID, string) (*entities.RConverter, error) {
	return s.getResult, s.getErr
}

func (s *converterRepositoryTestStub) GetByIdentityIncludingDeleted(context.Context, uuid.UUID, string) (*entities.RConverter, error) {
	return s.getIncludingDeletedResult, s.getIncludingDeletedErr
}

func (s *converterRepositoryTestStub) List(context.Context, ports.ConvertersFilter) ([]*entities.RConverter, error) {
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
