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

func (fakeCommits) Plan(ctx context.Context) (*commits.Plan, error) {
	if _, ok := entities.WorkspaceAccessFromContext(ctx); !ok {
		return nil, domainerrors.Internal("no_scope", "workspace scope missing")
	}
	return &commits.Plan{HeadSequence: 7, RevisionCount: 2, DocumentCount: 1}, nil
}

func (f *fakeCommits) Create(_ context.Context, message, policy string, expected int64) (*entities.Commit, error) {
	if expected != 7 {
		return nil, domainerrors.Conflict("head_sequence_conflict", "Workspace head changed")
	}
	f.created = append(f.created, message+"/"+policy)
	return &entities.Commit{ID: "c-1", HeadSequence: 7, Message: message}, nil
}

type fakeRevisions struct{ last ports.RevisionListOptions }

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

func newFakeUseCases() (UseCases, *fakeDocuments, *fakeCommits, *fakeRevisions) {
	queries := newFakeDocuments()
	commitUseCase := &fakeCommits{}
	revisionUseCase := &fakeRevisions{}
	return UseCases{
		Session: fakeSession{}, Workspaces: fakeWorkspaces{}, Domain: fakeDomain{}, Revisions: revisionUseCase, Commits: commitUseCase,
		Documents: map[string]DocumentUseCase{entities.CollectionQueries: queries},
	}, queries, commitUseCase, revisionUseCase
}
