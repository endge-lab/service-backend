package shared

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// TestCommitContext проверяет, что токен коммитит только с флагом commit и только
// от имени Admin/Platform Admin этого workspace, а сессия требует роль Admin.
func TestCommitContext(t *testing.T) {
	writeCommit := &entities.AccessTokenPrincipal{TokenID: "t", Scope: entities.AccessTokenScopeWrite, CanCommit: true}
	cases := map[string]struct {
		token     *entities.AccessTokenPrincipal
		role      string
		ownerRole string
		want      string
	}{
		"session admin":             {role: "admin"},
		"session platform admin":    {role: "platform_admin"},
		"session editor":            {role: "editor", want: "workspace_admin_required"},
		"read token":                {token: &entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeRead}, role: "viewer", ownerRole: "admin", want: "access_token_scope_insufficient"},
		"write token without flag":  {token: &entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite}, role: "editor", ownerRole: "admin", want: "access_token_commit_forbidden"},
		"commit token owner editor": {token: writeCommit, role: "editor", ownerRole: "editor", want: "access_token_owner_admin_required"},
		"commit token owner viewer": {token: writeCommit, role: "viewer", ownerRole: "viewer", want: "workspace_editor_required"},
		"commit token owner admin":  {token: writeCommit, role: "editor", ownerRole: "admin"},
		"commit token platform":     {token: writeCommit, role: "editor", ownerRole: "platform_admin"},
	}
	for name, tc := range cases {
		ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "u-1"}})
		ctx = entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "w-1"}, Role: tc.role, OwnerRole: tc.ownerRole})
		if tc.token != nil {
			ctx = entities.WithAccessToken(ctx, *tc.token)
		}
		_, _, err := CommitContext(ctx)
		if tc.want == "" {
			if err != nil {
				t.Errorf("%s: unexpected error %v", name, err)
			}
			continue
		}
		if got := domainerrors.CodeOf(err); got != tc.want {
			t.Errorf("%s: code %q, want %q (err %v)", name, got, tc.want, err)
		}
		if domainerrors.HTTPStatusOf(err) != 403 {
			t.Errorf("%s: status %d, want 403", name, domainerrors.HTTPStatusOf(err))
		}
	}
}

// TestAdminContextRejectsAccessTokens проверяет, что административные операции
// недоступны токену даже при роли Admin владельца, а сессии нужна роль Admin.
func TestAdminContextRejectsAccessTokens(t *testing.T) {
	cases := map[string]struct {
		token *entities.AccessTokenPrincipal
		role  string
		want  string
	}{
		"session admin":  {role: "admin"},
		"session editor": {role: "editor", want: "workspace_admin_required"},
		"write token":    {token: &entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite, CanCommit: true}, role: "admin", want: "access_token_admin_forbidden"},
	}
	for name, tc := range cases {
		ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "u-1"}})
		ctx = entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "w-1"}, Role: tc.role, OwnerRole: "admin"})
		if tc.token != nil {
			ctx = entities.WithAccessToken(ctx, *tc.token)
		}
		_, _, err := AdminContext(ctx)
		if tc.want == "" {
			if err != nil {
				t.Errorf("%s: unexpected error %v", name, err)
			}
			continue
		}
		if got := domainerrors.CodeOf(err); got != tc.want {
			t.Errorf("%s: code %q, want %q (err %v)", name, got, tc.want, err)
		}
		if domainerrors.HTTPStatusOf(err) != 403 {
			t.Errorf("%s: status %d, want 403", name, domainerrors.HTTPStatusOf(err))
		}
	}
}
