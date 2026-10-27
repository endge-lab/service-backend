package commits

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type fakeCommitRepository struct {
	ports.CommitRepository
	pending []entities.Revision
	created []entities.Commit
}

func (f *fakeCommitRepository) LatestCommit(context.Context, string) (*entities.Commit, error) {
	return &entities.Commit{ID: "c-0", HeadSequence: 5}, nil
}

func (f *fakeCommitRepository) PendingRevisions(context.Context, string, int64) ([]entities.Revision, error) {
	return f.pending, nil
}

func (f *fakeCommitRepository) SquashPending(context.Context, string, int64, string, string) ([]entities.Revision, error) {
	return f.pending, nil
}

func (f *fakeCommitRepository) CreateCommit(_ context.Context, value entities.Commit, _ []entities.CommitChange) (*entities.Commit, error) {
	f.created = append(f.created, value)
	return &value, nil
}

func (f *fakeCommitRepository) AttachRevisionsToCommit(context.Context, string, []string) error {
	return nil
}

type fakeRevisionRepository struct{ ports.RevisionRepository }

func (fakeRevisionRepository) CreateMutationBatch(context.Context, *string, string, string) (string, error) {
	return "b-1", nil
}

type fakeTx struct{ ports.TxManager }

func (fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// TestCreateWithAccessToken проверяет коммит токеном: флаг commit и роль Admin
// владельца обязательны; общий squash доступен, как у самого владельца.
func TestCreateWithAccessToken(t *testing.T) {
	commitToken := entities.AccessTokenPrincipal{TokenID: "t-1", WorkspaceID: "w-1", Scope: entities.AccessTokenScopeWrite, CanCommit: true}
	cases := map[string]struct {
		token     entities.AccessTokenPrincipal
		ownerRole string
		policy    string
		want      string
	}{
		"owner admin preserve":      {token: commitToken, ownerRole: "admin", policy: entities.RevisionPolicyPreserve},
		"owner admin shared squash": {token: commitToken, ownerRole: "admin", policy: entities.RevisionPolicySquash},
		"owner platform admin":      {token: commitToken, ownerRole: "platform_admin", policy: entities.RevisionPolicyPreserve},
		"owner editor":              {token: commitToken, ownerRole: "editor", policy: entities.RevisionPolicyPreserve, want: "access_token_owner_admin_required"},
		"without commit flag":       {token: entities.AccessTokenPrincipal{TokenID: "t-2", WorkspaceID: "w-1", Scope: entities.AccessTokenScopeWrite}, ownerRole: "admin", policy: entities.RevisionPolicyPreserve, want: "access_token_commit_forbidden"},
	}
	for name, tc := range cases {
		repository := &fakeCommitRepository{pending: []entities.Revision{{ID: "r-1", DocumentType: "queries", DocumentID: "d-1", CreatedBy: entities.Actor{ID: "someone-else"}}}}
		useCase := NewUseCase(repository, fakeRevisionRepository{}, fakeTx{}, nil)
		ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "owner"}})
		ctx = entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "w-1", HeadSequence: 7}, Role: "editor", OwnerRole: tc.ownerRole})
		ctx = entities.WithAccessToken(ctx, tc.token)
		created, err := useCase.Create(ctx, "агент", tc.policy, 7)
		if tc.want == "" {
			if err != nil || created == nil || len(repository.created) != 1 || repository.created[0].CreatedBy.ID != "owner" {
				t.Errorf("%s: created %+v err %v", name, created, err)
			}
			continue
		}
		if domainerrors.CodeOf(err) != tc.want || domainerrors.HTTPStatusOf(err) != 403 || len(repository.created) != 0 {
			t.Errorf("%s: err %v, want %s", name, err, tc.want)
		}
	}
}
