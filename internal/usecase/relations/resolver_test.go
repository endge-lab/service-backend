package relations

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

func TestResolveProjectNormalizesAndScopesLookup(t *testing.T) {
	workspaceID := uuid.New()
	projects := &projectsStub{project: &entities.RProject{ID: uuid.New(), WorkspaceID: workspaceID, Identity: "demo"}}
	resolver := NewResolver(projects, nil)

	project, err := resolver.ResolveProject(entities.WithWorkspaceID(context.Background(), workspaceID), workspaceID, " demo ")
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != projects.project.ID || projects.identity != "demo" {
		t.Fatalf("project = %#v, lookup identity = %q", project, projects.identity)
	}
}

func TestResolveProjectRejectsUnscopedAndMissingRelations(t *testing.T) {
	workspaceID := uuid.New()
	resolver := NewResolver(&projectsStub{err: apperrors.NotFound("not_found", "project not found")}, nil)

	if _, err := resolver.ResolveProject(context.Background(), workspaceID, "demo"); apperrors.CodeOf(err) != "workspace_required" {
		t.Fatalf("unscoped error code = %q", apperrors.CodeOf(err))
	}
	if _, err := resolver.ResolveProject(entities.WithWorkspaceID(context.Background(), workspaceID), workspaceID, "missing"); apperrors.CodeOf(err) != "project_not_found" {
		t.Fatalf("not found error code = %q", apperrors.CodeOf(err))
	}
	if _, err := resolver.ResolveProject(entities.WithWorkspaceID(context.Background(), workspaceID), uuid.New(), "demo"); apperrors.CodeOf(err) != "workspace_scope_mismatch" {
		t.Fatalf("mismatch error code = %q", apperrors.CodeOf(err))
	}
}

func TestResolveFolderValidatesScopeTypeAndProject(t *testing.T) {
	workspaceID := uuid.New()
	projectID := uuid.New()
	ctx := entities.WithWorkspaceID(context.Background(), workspaceID)
	project := projectID
	base := &entities.RFolder{ID: uuid.New(), WorkspaceID: workspaceID, ProjectID: &project, EntityType: entities.FolderEntityTypeQueries, Identity: "root-queries"}

	for _, tc := range []struct {
		name     string
		folder   *entities.RFolder
		wantCode string
	}{
		{name: "workspace mismatch", folder: &entities.RFolder{WorkspaceID: uuid.New(), ProjectID: &project, EntityType: entities.FolderEntityTypeQueries}, wantCode: "folder_workspace_mismatch"},
		{name: "entity type mismatch", folder: &entities.RFolder{WorkspaceID: workspaceID, ProjectID: &project, EntityType: entities.FolderEntityTypeConverters}, wantCode: "folder_entity_type_mismatch"},
		{name: "project mismatch", folder: &entities.RFolder{WorkspaceID: workspaceID, ProjectID: uuidPtr(uuid.New()), EntityType: entities.FolderEntityTypeQueries}, wantCode: "folder_project_mismatch"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resolver := NewResolver(nil, &foldersStub{folder: tc.folder})
			_, err := resolver.ResolveFolder(ctx, workspaceID, "root-queries", entities.FolderEntityTypeQueries, &projectID)
			if code := apperrors.CodeOf(err); code != tc.wantCode {
				t.Fatalf("error code = %q, want %q", code, tc.wantCode)
			}
		})
	}

	resolver := NewResolver(nil, &foldersStub{folder: base})
	if _, err := resolver.ResolveFolder(ctx, workspaceID, "root-queries", entities.FolderEntityTypeQueries, &projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewResolver(nil, &foldersStub{err: apperrors.NotFound("not_found", "folder not found")}).ResolveFolder(ctx, workspaceID, "missing", entities.FolderEntityTypeQueries, &projectID); apperrors.CodeOf(err) != "folder_not_found" {
		t.Fatalf("not found error code = %q", apperrors.CodeOf(err))
	}
	storageErr := stderrors.New("storage unavailable")
	if _, err := NewResolver(nil, &foldersStub{err: storageErr}).ResolveFolder(ctx, workspaceID, "root-queries", entities.FolderEntityTypeQueries, &projectID); !stderrors.Is(err, storageErr) {
		t.Fatalf("error = %v, want storage error", err)
	}
}

func uuidPtr(value uuid.UUID) *uuid.UUID { return &value }

type projectsStub struct {
	ports.ProjectsRepository
	project  *entities.RProject
	err      error
	identity string
}

func (s *projectsStub) GetByIdentity(_ context.Context, identity string) (*entities.RProject, error) {
	s.identity = identity
	return s.project, s.err
}

type foldersStub struct {
	ports.FoldersRepository
	folder *entities.RFolder
	err    error
}

func (s *foldersStub) GetByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	return s.folder, s.err
}
