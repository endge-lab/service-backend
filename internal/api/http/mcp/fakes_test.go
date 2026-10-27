package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/commits"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/facets"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/session"
)

// Поддельные use case повторяют только то поведение, которое видит агент:
// проверку workspace токена, scope записи, конфликт revision и отметку изменений токеном.

var testUser = &entities.User{ID: "u-1", Username: "egor", DisplayName: "Егор", Active: true}

type fakeWorkspaces struct{}

func (fakeWorkspaces) Authorize(ctx context.Context, identity string) (entities.WorkspaceAccess, error) {
	if identity != "afl" && identity != "kzn" {
		return entities.WorkspaceAccess{}, domainerrors.NotFound("workspace_not_found", "Workspace not found")
	}
	role, ownerRole := "editor", ""
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		if !strings.EqualFold(token.WorkspaceIdentity, identity) {
			return entities.WorkspaceAccess{}, domainerrors.Forbidden("access_token_workspace_forbidden", "Access token is issued for another workspace")
		}
		// Владелец токенов — Admin workspace, кроме t-full-editor (Editor).
		ownerRole = "admin"
		if token.TokenID == "t-full-editor" {
			ownerRole = "editor"
		}
		role = token.CapWorkspaceRole(ownerRole)
	}
	return entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "ws-" + identity, Identity: identity, DisplayName: strings.ToUpper(identity)}, Role: role, OwnerRole: ownerRole}, nil
}

// testAdminUser — Admin workspace afl; сессия JWT такого пользователя видит endge_commit.
var testAdminUser = &entities.User{ID: "u-admin", Username: "admin", DisplayName: "Админ", Active: true}

// userWorkspaces — workspace JWT-сессии: testUser — Editor/Viewer, testAdminUser — Admin в afl.
func userWorkspaces(user *entities.User) []entities.WorkspaceAccess {
	role := "editor"
	if user != nil && user.ID == testAdminUser.ID {
		role = "admin"
	}
	return []entities.WorkspaceAccess{{Workspace: entities.Workspace{Identity: "afl", DisplayName: "АФЛ"}, Role: role}, {Workspace: entities.Workspace{Identity: "kzn", DisplayName: "КЗН"}, Role: "viewer"}}
}

func (fakeWorkspaces) ListAccess(ctx context.Context) ([]entities.WorkspaceAccess, error) {
	actor, _ := entities.CurrentActorFromContext(ctx)
	return userWorkspaces(actor.User), nil
}

type fakeSession struct{}

func (fakeSession) Current(ctx context.Context) (*session.Result, error) {
	actor, _ := entities.CurrentActorFromContext(ctx)
	items := userWorkspaces(actor.User)
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		items = []entities.WorkspaceAccess{{Workspace: entities.Workspace{Identity: token.WorkspaceIdentity}, Role: token.CapWorkspaceRole("editor")}}
	}
	return &session.Result{User: actor.User, PlatformAdmin: actor.PlatformAdmin, Workspaces: items}, nil
}

type fakeDomain struct{}

func (fakeDomain) Live(ctx context.Context) (json.RawMessage, error) {
	scope, _ := entities.WorkspaceAccessFromContext(ctx)
	return json.RawMessage(`{"domainVersion":"dv2:x","workspace":{"identity":"` + scope.Workspace.Identity + `","displayName":"АФЛ","secret":"x"},"documents":{"queries":[{"identity":"flights","displayName":"Рейсы","source":"defineQuery({})","state":{"revision":3}}]}}`), nil
}

type fakeCommits struct{ created []string }

func (fakeCommits) PlanWithDocuments(ctx context.Context) (*commits.Plan, error) {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return nil, domainerrors.Internal("no_scope", "workspace scope missing")
	}
	return &commits.Plan{HeadSequence: 7, RevisionCount: 2, DocumentCount: 1, Documents: []commits.PlanDocument{
		{Collection: "queries", Identity: "flights", DisplayName: "Рейсы", RevisionCount: 2, Operations: []string{"update"}, Authors: []entities.Actor{{ID: "u-1"}}, AccessTokens: []commits.PlanAccessToken{{ID: "t-1", Name: "Codex"}}},
	}}, nil
}

func fakeCommitList() []entities.Commit {
	before, after := "r-1", "r-2"
	return []entities.Commit{
		{ID: "c-2", HeadSequence: 7, Message: "второй", CreatedAt: time.Unix(20, 0).UTC(), Changes: []entities.CommitChange{{DocumentType: "queries", DocumentIdentity: "flights", BeforeRevisionID: &before, AfterRevisionID: &after, Operation: "update"}}},
		{ID: "c-1", HeadSequence: 3, Message: "первый", CreatedAt: time.Unix(10, 0).UTC()},
	}
}

func (fakeCommits) List(ctx context.Context) ([]entities.Commit, error) {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return nil, domainerrors.Internal("no_scope", "workspace scope missing")
	}
	return fakeCommitList(), nil
}

func (fakeCommits) Get(_ context.Context, id string) (*entities.Commit, error) {
	for _, value := range fakeCommitList() {
		if value.ID == id {
			return &value, nil
		}
	}
	return nil, domainerrors.NotFound("commit_not_found", "Commit not found")
}

func (f *fakeCommits) Create(_ context.Context, message, policy string, expected int64) (*entities.Commit, error) {
	if expected != 7 {
		return nil, domainerrors.Conflict("head_sequence_conflict", "Workspace head changed")
	}
	f.created = append(f.created, message+"/"+policy)
	return &entities.Commit{ID: "c-1", HeadSequence: 7, Message: message}, nil
}

type fakeRevisions struct {
	last     ports.RevisionListOptions
	restored []string
}

func (f *fakeRevisions) Get(_ context.Context, collection, identity, revisionID string) (*entities.Revision, error) {
	if revisionID != "r-1" {
		return nil, domainerrors.NotFound("revision_not_found", "Revision not found")
	}
	return &entities.Revision{ID: "r-1", DocumentType: collection, DocumentIdentity: identity, RevisionNumber: 1, Operation: "create", Snapshot: json.RawMessage(`{"displayName":"Рейсы","data":{"source":"old"}}`)}, nil
}

// Restore повторяет поведение use case: запись (scope токена) и проверка revision документа.
func (f *fakeRevisions) Restore(ctx context.Context, collection, identity, revisionID string, expected int) (*entities.Document, error) {
	if token, ok := entities.AccessTokenFromContext(ctx); ok && !token.CanWrite() {
		return nil, domainerrors.Forbidden("access_token_scope_insufficient", "Access token is read-only")
	}
	if expected != 1 {
		return nil, domainerrors.Conflict("revision_conflict", "Revision conflict")
	}
	f.restored = append(f.restored, collection+"/"+identity+"@"+revisionID)
	return &entities.Document{ID: "d-1", Identity: identity, DisplayName: "Рейсы", Revision: 2, Active: true, Data: json.RawMessage(`{"source":"old","sourceVersion":2}`), Meta: json.RawMessage(`{}`)}, nil
}

func (f *fakeRevisions) List(_ context.Context, collection, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	f.last = options
	tokenID, tokenName := "t-1", "Codex"
	return ports.RevisionPage{Total: 2, Items: []entities.Revision{{ID: "r-2", DocumentType: collection, DocumentIdentity: identity, RevisionNumber: 2, Operation: "update", AccessTokenID: &tokenID, AccessTokenName: &tokenName, CreatedAt: time.Unix(0, 0).UTC(), Snapshot: json.RawMessage(`{"source":"x"}`)}}}, nil
}

// fakeDocuments — коллекция в памяти; marks хранит id токена из context каждой записи.
type fakeDocuments struct {
	mu    sync.Mutex
	items map[string]entities.Document
	marks []string
}

func newFakeDocuments() *fakeDocuments {
	return &fakeDocuments{items: map[string]entities.Document{
		"flights": {ID: "d-1", Identity: "flights", DisplayName: "Рейсы", Revision: 1, Active: true, Data: json.RawMessage(`{"source":"defineQuery({})","sourceVersion":2}`), Meta: json.RawMessage(`{}`)},
	}}
}

func (f *fakeDocuments) write(ctx context.Context) error {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return domainerrors.Internal("no_scope", "workspace scope missing")
	}
	mark := "user"
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		if !token.CanWrite() {
			return domainerrors.Forbidden("access_token_scope_insufficient", "Access token is read-only")
		}
		mark = token.TokenID
	}
	f.marks = append(f.marks, mark)
	return nil
}

func (f *fakeDocuments) List(ctx context.Context, filter ports.DocumentFilter) ([]entities.Document, error) {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return nil, domainerrors.Internal("no_scope", "workspace scope missing")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	result := []entities.Document{}
	for _, item := range f.items {
		result = append(result, item)
	}
	return result, nil
}

func (f *fakeDocuments) Get(_ context.Context, identity string, _ bool) (*entities.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[identity]
	if !ok {
		return nil, domainerrors.NotFound("document_not_found", "Document not found")
	}
	return &item, nil
}

func (f *fakeDocuments) Create(ctx context.Context, _ documents.CreateInput) (*entities.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.write(ctx); err != nil {
		return nil, err
	}
	item := entities.Document{ID: "d-new", Identity: "created", DisplayName: "Создан", Revision: 1, Active: true, Data: json.RawMessage(`{}`), Meta: json.RawMessage(`{}`)}
	f.items[item.Identity] = item
	return &item, nil
}

func (f *fakeDocuments) Patch(ctx context.Context, identity string, _ documents.PatchInput, expected int) (*entities.Document, error) {
	return f.bump(ctx, identity, expected, false)
}

func (f *fakeDocuments) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return f.bump(ctx, identity, expected, true)
}

func (f *fakeDocuments) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return f.bump(ctx, identity, expected, false)
}

func (f *fakeDocuments) bump(ctx context.Context, identity string, expected int, deleted bool) (*entities.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.write(ctx); err != nil {
		return nil, err
	}
	item, ok := f.items[identity]
	if !ok {
		return nil, domainerrors.NotFound("document_not_found", "Document not found")
	}
	if item.Revision != expected {
		return nil, domainerrors.WithDetails(domainerrors.Conflict("revision_conflict", "Revision conflict"), map[string]any{"currentRevision": item.Revision})
	}
	item.Revision++
	item.Active = !deleted
	f.items[identity] = item
	return &item, nil
}

// fakeFacets — фасет tenant с документами tenant-a; calls фиксирует вызовы use case.
type fakeFacets struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeFacets) record(ctx context.Context, call string, write bool) error {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return domainerrors.Internal("no_scope", "workspace scope missing")
	}
	if token, ok := entities.AccessTokenFromContext(ctx); ok && write && !token.CanWrite() {
		return domainerrors.Forbidden("access_token_scope_insufficient", "Access token is read-only")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
	return nil
}

func facetDocument(revision int) *entities.Document {
	return &entities.Document{ID: "f-1", Type: entities.CollectionFacets, Identity: "tenant", DisplayName: "Тенант", Revision: revision, Active: true, Data: json.RawMessage(`{"icon":"Building","color":"#2563eb","position":1,"documentCount":1}`), Meta: json.RawMessage(`{}`)}
}

func facetEntry(identity string, revision int) *entities.Document {
	return &entities.Document{ID: "fd-1", Type: entities.CollectionFacetDocuments, Identity: identity, DisplayName: "Тенант A", Revision: revision, Active: true, Data: json.RawMessage(`{"facetIdentity":"tenant","configuration":{"region":"eu"}}`), Meta: json.RawMessage(`{}`)}
}

func facetRevisions() []entities.Revision {
	return []entities.Revision{{ID: "fr-2", DocumentType: "facet-documents", DocumentIdentity: "tenant-a", RevisionNumber: 2, Operation: "update", Snapshot: json.RawMessage(`{}`)}, {ID: "fr-1", DocumentType: "facet-documents", DocumentIdentity: "tenant-a", RevisionNumber: 1, Operation: "create", Snapshot: json.RawMessage(`{}`)}}
}

func (f *fakeFacets) List(ctx context.Context, _ bool) ([]entities.Document, error) {
	return []entities.Document{*facetDocument(1)}, f.record(ctx, "List", false)
}

func (f *fakeFacets) Get(ctx context.Context, identity string, _ bool) (*entities.Document, error) {
	return facetDocument(1), f.record(ctx, "Get "+identity, false)
}

func (f *fakeFacets) Create(ctx context.Context, input facets.FacetCreateInput) (*entities.Document, error) {
	return facetDocument(1), f.record(ctx, "Create "+input.Identity+" "+input.Color, true)
}

func (f *fakeFacets) Patch(ctx context.Context, identity string, input facets.FacetPatchInput, expected int) (*entities.Document, error) {
	return facetDocument(expected + 1), f.record(ctx, "Patch "+identity, true)
}

func (f *fakeFacets) Delete(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return facetDocument(expected + 1), f.record(ctx, "Delete "+identity, true)
}

func (f *fakeFacets) Restore(ctx context.Context, identity string, expected int) (*entities.Document, error) {
	return facetDocument(expected + 1), f.record(ctx, "Restore "+identity, true)
}

func (f *fakeFacets) ListRevisions(ctx context.Context, identity string) ([]entities.Revision, error) {
	return facetRevisions(), f.record(ctx, "ListRevisions "+identity, false)
}

func (f *fakeFacets) GetRevision(ctx context.Context, identity, revisionID string) (*entities.Revision, error) {
	return &facetRevisions()[1], f.record(ctx, "GetRevision "+identity+" "+revisionID, false)
}

func (f *fakeFacets) RestoreRevision(ctx context.Context, identity, revisionID string, expected int) (*entities.Document, error) {
	return facetDocument(expected + 1), f.record(ctx, "RestoreRevision "+identity+" "+revisionID, true)
}

func (f *fakeFacets) ListDocuments(ctx context.Context, facet string, _ ports.DocumentFilter) ([]entities.Document, error) {
	return []entities.Document{*facetEntry("tenant-a", 2)}, f.record(ctx, "ListDocuments "+facet, false)
}

func (f *fakeFacets) GetDocument(ctx context.Context, facet, identity string, _ bool) (*entities.Document, error) {
	return facetEntry(identity, 2), f.record(ctx, "GetDocument "+facet+" "+identity, false)
}

func (f *fakeFacets) CreateDocument(ctx context.Context, facet string, input facets.FacetDocumentCreateInput) (*entities.Document, error) {
	return facetEntry(input.Identity, 1), f.record(ctx, "CreateDocument "+facet+" "+input.Identity, true)
}

func (f *fakeFacets) PatchDocument(ctx context.Context, facet, identity string, _ facets.FacetDocumentPatchInput, expected int) (*entities.Document, error) {
	if expected != 2 {
		return nil, domainerrors.Conflict("revision_conflict", "Revision conflict")
	}
	return facetEntry(identity, expected+1), f.record(ctx, "PatchDocument "+facet+" "+identity, true)
}

func (f *fakeFacets) DeleteDocument(ctx context.Context, facet, identity string, expected int) (*entities.Document, error) {
	return facetEntry(identity, expected+1), f.record(ctx, "DeleteDocument "+facet+" "+identity, true)
}

func (f *fakeFacets) RestoreDocument(ctx context.Context, facet, identity string, expected int) (*entities.Document, error) {
	return facetEntry(identity, expected+1), f.record(ctx, "RestoreDocument "+facet+" "+identity, true)
}

func (f *fakeFacets) ListDocumentRevisions(ctx context.Context, facet, identity string) ([]entities.Revision, error) {
	return facetRevisions(), f.record(ctx, "ListDocumentRevisions "+facet+" "+identity, false)
}

func (f *fakeFacets) GetDocumentRevision(ctx context.Context, facet, identity, revisionID string) (*entities.Revision, error) {
	return &facetRevisions()[1], f.record(ctx, "GetDocumentRevision "+facet+" "+identity+" "+revisionID, false)
}

func (f *fakeFacets) RestoreDocumentRevision(ctx context.Context, facet, identity, revisionID string, expected int) (*entities.Document, error) {
	return facetEntry(identity, expected+1), f.record(ctx, "RestoreDocumentRevision "+facet+" "+identity+" "+revisionID, true)
}

type fakes struct {
	documents *fakeDocuments
	commits   *fakeCommits
	revisions *fakeRevisions
	facets    *fakeFacets
}

func newFakeUseCases() (UseCases, fakes) {
	f := fakes{documents: newFakeDocuments(), commits: &fakeCommits{}, revisions: &fakeRevisions{}, facets: &fakeFacets{}}
	return UseCases{
		Session: fakeSession{}, Workspaces: fakeWorkspaces{}, Domain: fakeDomain{}, Revisions: f.revisions, Commits: f.commits, Facets: f.facets,
		Documents: map[string]DocumentUseCase{entities.CollectionQueries: f.documents},
	}, f
}
