package data_views

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

func TestCreateReturnsResolvedRelations(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-data-views"}
	query := &entities.Query{ID: uuid.New(), ProjectID: project.ID, Identity: "users-list"}
	repository := &dataViewsRepositoryStub{createResult: &entities.DataView{ID: uuid.New(), Identity: "users-table", FolderID: folder.ID, QueryID: query.ID}}
	service := newDataViewServiceForTest(project, &dataViewFoldersRepositoryStub{folder: folder}, &dataViewQueriesRepositoryStub{query: query}, repository)

	result, err := service.Create(context.Background(), adapters.CreateDataViewInput{ProjectIdentity: "demo", FolderIdentity: folder.Identity, QueryIdentity: query.Identity, Identity: "users-table", DisplayName: "Users Table", ViewType: "pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	if result.FolderIdentity != folder.Identity || result.QueryIdentity != query.Identity || repository.created.QueryID != query.ID || repository.created.Source == nil {
		t.Fatalf("result = %#v, created = %#v", result, repository.created)
	}
}

func TestCreateRejectsQueryFromAnotherProject(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-data-views"}
	queries := &dataViewQueriesRepositoryStub{getErr: apperrors.NotFound("not_found", "query not found"), existsOutside: true}
	service := newDataViewServiceForTest(project, &dataViewFoldersRepositoryStub{folder: folder}, queries, &dataViewsRepositoryStub{})

	_, err := service.Create(context.Background(), adapters.CreateDataViewInput{ProjectIdentity: "demo", FolderIdentity: folder.Identity, QueryIdentity: "foreign-query", Identity: "users-table", DisplayName: "Users Table", ViewType: "pipeline"})
	if got := apperrors.CodeOf(err); got != "query_project_mismatch" {
		t.Fatalf("code = %q, want query_project_mismatch", got)
	}
}

func TestListResolvesRelationsWithoutNPlusOne(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	folder := &entities.Folder{ID: uuid.New(), Identity: "root-data-views"}
	query := &entities.Query{ID: uuid.New(), ProjectID: project.ID, Identity: "users-list"}
	folders := &dataViewFoldersRepositoryStub{folders: []*entities.Folder{folder}}
	queries := &dataViewQueriesRepositoryStub{listResult: []*entities.Query{query}}
	dataViews := &dataViewsRepositoryStub{listResult: []*entities.DataView{{ID: uuid.New(), FolderID: folder.ID, QueryID: query.ID, Identity: "users-table"}}}
	service := newDataViewServiceForTest(project, folders, queries, dataViews)

	result, err := service.List(context.Background(), adapters.ListDataViewsInput{ProjectIdentity: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].FolderIdentity != folder.Identity || result[0].QueryIdentity != query.Identity || folders.listCalls != 1 || queries.listCalls != 1 {
		t.Fatalf("result = %#v, folder list calls = %d, query list calls = %d", result, folders.listCalls, queries.listCalls)
	}
}

func TestCreateRejectsFolderWithWrongEntityType(t *testing.T) {
	project := &entities.Project{ID: uuid.New(), Identity: "demo"}
	service := newDataViewServiceForTest(project, &dataViewFoldersRepositoryStub{getErr: apperrors.NotFound("not_found", "folder not found")}, &dataViewQueriesRepositoryStub{}, &dataViewsRepositoryStub{})
	_, err := service.Create(context.Background(), adapters.CreateDataViewInput{ProjectIdentity: "demo", FolderIdentity: "other", QueryIdentity: "users-list", Identity: "users-table", DisplayName: "Users Table", ViewType: "pipeline"})
	if got := apperrors.CodeOf(err); got != "folder_entity_type_mismatch" {
		t.Fatalf("code = %q, want folder_entity_type_mismatch", got)
	}
}

func newDataViewServiceForTest(project *entities.Project, folders *dataViewFoldersRepositoryStub, queries *dataViewQueriesRepositoryStub, dataViews *dataViewsRepositoryStub) *DataView {
	return NewDataViewService(DataViewParams{ProjectRepository: &dataViewProjectsRepositoryStub{project: project}, FolderRepository: folders, QueryRepository: queries, DataViewRepository: dataViews, Tracer: otel.Tracer("data-views-test"), Logger: zap.NewNop()})
}

type dataViewProjectsRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.Project
}

func (s *dataViewProjectsRepositoryStub) GetByIdentity(context.Context, string) (*entities.Project, error) {
	return s.project, nil
}

type dataViewFoldersRepositoryStub struct {
	ports.FoldersRepository
	folder    *entities.Folder
	getErr    error
	folders   []*entities.Folder
	listCalls int
}

func (s *dataViewFoldersRepositoryStub) GetByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.Folder, error) {
	return s.folder, s.getErr
}
func (s *dataViewFoldersRepositoryStub) List(context.Context, *uuid.UUID, entities.FolderEntityType) ([]*entities.Folder, error) {
	s.listCalls++
	return s.folders, nil
}

type dataViewQueriesRepositoryStub struct {
	ports.QueriesRepository
	query         *entities.Query
	getErr        error
	existsOutside bool
	listResult    []*entities.Query
	listCalls     int
}

func (s *dataViewQueriesRepositoryStub) GetByIdentity(context.Context, uuid.UUID, string) (*entities.Query, error) {
	return s.query, s.getErr
}
func (s *dataViewQueriesRepositoryStub) ExistsActiveByIdentityOutsideProject(context.Context, uuid.UUID, string) (bool, error) {
	return s.existsOutside, nil
}
func (s *dataViewQueriesRepositoryStub) List(context.Context, ports.QueriesFilter) ([]*entities.Query, error) {
	s.listCalls++
	return s.listResult, nil
}

type dataViewsRepositoryStub struct {
	ports.DataViewsRepository
	created      *entities.DataView
	createResult *entities.DataView
	listResult   []*entities.DataView
}

func (s *dataViewsRepositoryStub) ExistsByIdentity(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
func (s *dataViewsRepositoryStub) Create(_ context.Context, dataView *entities.DataView) (*entities.DataView, error) {
	s.created = dataView
	return s.createResult, nil
}
func (s *dataViewsRepositoryStub) List(context.Context, ports.DataViewsFilter) ([]*entities.DataView, error) {
	return s.listResult, nil
}
