package queries

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestCreateNormalizesPayloadAndReturnsFolderIdentity(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-queries"}
	repository := &queriesRepositoryStub{createResult: &entities.RQuery{ID: uuid.New(), Identity: "users-list", FolderID: folder.ID}}
	service := newQueryServiceForTest(project, &queryFoldersRepositoryStub{folder: folder}, repository)

	result, err := service.Create(context.Background(), CreateQueryInput{ProjectIdentity: " demo ", FolderIdentity: " root-queries ", Identity: " users-list ", DisplayName: " Users ", QueryType: " http "})
	if err != nil {
		t.Fatal(err)
	}
	if result.FolderIdentity != folder.Identity || repository.created.Source == nil || repository.created.Params == nil || repository.created.Headers == nil || repository.created.Meta == nil {
		t.Fatalf("result = %#v, created = %#v", result, repository.created)
	}
}

func TestCreateRejectsConflictAndFolderEntityTypeMismatch(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	folder := &entities.RFolder{ID: uuid.New(), Identity: "root-queries"}
	input := CreateQueryInput{ProjectIdentity: "demo", FolderIdentity: folder.Identity, Identity: "users-list", DisplayName: "Users", QueryType: "http"}

	service := newQueryServiceForTest(project, &queryFoldersRepositoryStub{folder: folder}, &queriesRepositoryStub{exists: true})
	if got := apperrors.CodeOf(mustCreateError(t, service, input)); got != "identity_conflict" {
		t.Fatalf("code = %q", got)
	}

	service = newQueryServiceForTest(project, &queryFoldersRepositoryStub{getErr: apperrors.NotFound("not_found", "folder not found")}, &queriesRepositoryStub{})
	if got := apperrors.CodeOf(mustCreateError(t, service, input)); got != "folder_entity_type_mismatch" {
		t.Fatalf("code = %q", got)
	}
}

func TestListResolvesFoldersWithoutNPlusOne(t *testing.T) {
	project := &entities.RProject{ID: uuid.New(), Identity: "demo"}
	first := &entities.RFolder{ID: uuid.New(), Identity: "root-queries"}
	second := &entities.RFolder{ID: uuid.New(), Identity: "api"}
	folders := &queryFoldersRepositoryStub{folders: []*entities.RFolder{first, second}}
	repository := &queriesRepositoryStub{listResult: []*entities.RQuery{{ID: uuid.New(), FolderID: first.ID, Identity: "one"}, {ID: uuid.New(), FolderID: second.ID, Identity: "two"}}}
	service := newQueryServiceForTest(project, folders, repository)

	result, err := service.List(context.Background(), ListQueriesInput{ProjectIdentity: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].FolderIdentity != first.Identity || result[1].FolderIdentity != second.Identity || folders.listCalls != 1 {
		t.Fatalf("result = %#v, list calls = %d", result, folders.listCalls)
	}
}

func mustCreateError(t *testing.T, service *Query, input CreateQueryInput) error {
	t.Helper()
	_, err := service.Create(context.Background(), input)
	return err
}

func newQueryServiceForTest(project *entities.RProject, folders *queryFoldersRepositoryStub, queries *queriesRepositoryStub) *Query {
	return NewQueryService(QueryParams{ProjectRepository: &queryProjectsRepositoryStub{project: project}, FolderRepository: folders, QueryRepository: queries, Tracer: otel.Tracer("queries-test"), Logger: zap.NewNop()})
}

type queryProjectsRepositoryStub struct {
	ports.ProjectsRepository
	project *entities.RProject
}

func (s *queryProjectsRepositoryStub) GetByIdentity(context.Context, string) (*entities.RProject, error) {
	return s.project, nil
}

type queryFoldersRepositoryStub struct {
	ports.FoldersRepository
	folder    *entities.RFolder
	getErr    error
	folders   []*entities.RFolder
	listCalls int
}

func (s *queryFoldersRepositoryStub) GetByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	return s.folder, s.getErr
}
func (s *queryFoldersRepositoryStub) List(context.Context, *uuid.UUID, entities.FolderEntityType) ([]*entities.RFolder, error) {
	s.listCalls++
	return s.folders, nil
}

type queriesRepositoryStub struct {
	ports.QueriesRepository
	exists       bool
	created      *entities.RQuery
	createResult *entities.RQuery
	listResult   []*entities.RQuery
}

func (s *queriesRepositoryStub) ExistsByIdentity(context.Context, uuid.UUID, string) (bool, error) {
	return s.exists, nil
}
func (s *queriesRepositoryStub) Create(_ context.Context, query *entities.RQuery) (*entities.RQuery, error) {
	s.created = query
	return s.createResult, nil
}
func (s *queriesRepositoryStub) List(context.Context, ports.QueriesFilter) ([]*entities.RQuery, error) {
	return s.listResult, nil
}
