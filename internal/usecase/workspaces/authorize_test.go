package workspaces

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type roleWorkspaces struct {
	ports.WorkspaceRepository
	role string
}

func (r roleWorkspaces) GetWorkspace(context.Context, string) (*entities.Workspace, error) {
	return &entities.Workspace{ID: "w-1", Identity: "afl"}, nil
}

func (r roleWorkspaces) WorkspaceRole(_ context.Context, _, _ string, platformAdmin bool) (string, error) {
	if platformAdmin {
		return "platform_admin", nil
	}
	return r.role, nil
}

// TestAuthorizeKeepsTokenOwnerRole проверяет, что роль токена сужается,
// а исходная роль владельца сохраняется для проверки коммита.
func TestAuthorizeKeepsTokenOwnerRole(t *testing.T) {
	cases := map[string]struct {
		token               *entities.AccessTokenPrincipal
		role                string
		wantRole, wantOwner string
	}{
		"session admin":        {role: "admin", wantRole: "admin"},
		"write token admin":    {token: &entities.AccessTokenPrincipal{WorkspaceID: "w-1", Scope: entities.AccessTokenScopeWrite}, role: "admin", wantRole: "editor", wantOwner: "admin"},
		"read token editor":    {token: &entities.AccessTokenPrincipal{WorkspaceID: "w-1", Scope: entities.AccessTokenScopeRead}, role: "editor", wantRole: "viewer", wantOwner: "editor"},
		"token platform admin": {token: &entities.AccessTokenPrincipal{WorkspaceID: "w-1", Scope: entities.AccessTokenScopeWrite, OwnerPlatformAdmin: true}, wantRole: "editor", wantOwner: "platform_admin"},
	}
	for name, tc := range cases {
		ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "u-1"}})
		if tc.token != nil {
			ctx = entities.WithAccessToken(ctx, *tc.token)
		}
		access, err := NewUseCase(roleWorkspaces{role: tc.role}, nil, nil, nil, nil, nil).Authorize(ctx, "afl")
		if err != nil || access.Role != tc.wantRole || access.OwnerRole != tc.wantOwner {
			t.Errorf("%s: role %q owner %q err %v, want %q/%q", name, access.Role, access.OwnerRole, err, tc.wantRole, tc.wantOwner)
		}
	}
}

func TestTokenWorkspaceManagementUsesCurrentOwnerRole(t *testing.T) {
	actor := entities.CurrentActor{User: &entities.User{ID: "u-1"}}
	ctx := entities.WithCurrentActor(context.Background(), actor)
	token := entities.AccessTokenPrincipal{AllWorkspaces: true, ManageWorkspaces: true, Scope: entities.AccessTokenScopeWrite}
	managed := entities.WithAccessToken(ctx, token)
	for _, tc := range []struct {
		role, code string
	}{
		{role: "admin", code: "precondition_required"},
		{role: "editor", code: "workspace_admin_required"},
		{role: "viewer", code: "workspace_admin_required"},
		{role: "", code: "workspace_forbidden"},
	} {
		useCase := NewUseCase(roleWorkspaces{role: tc.role}, nil, nil, nil, nil, nil)
		if _, err := useCase.Delete(managed, "afl", 0); domainerrors.CodeOf(err) != tc.code {
			t.Errorf("archive with owner role %q: got %v, want %s", tc.role, err, tc.code)
		}
	}
	useCase := NewUseCase(roleWorkspaces{role: "admin"}, nil, nil, nil, nil, nil)
	if _, err := useCase.Create(managed, CreateInput{}); domainerrors.CodeOf(err) != "platform_admin_required" {
		t.Fatalf("only a platform admin can create: %v", err)
	}
	token.OwnerPlatformAdmin = true
	if _, err := useCase.Create(entities.WithAccessToken(ctx, token), CreateInput{}); domainerrors.CodeOf(err) != "identity_required" {
		t.Fatalf("platform admin's management token should reach input validation: %v", err)
	}
	token.ManageWorkspaces = false
	if _, err := useCase.Restore(entities.WithAccessToken(ctx, token), "afl", 1); domainerrors.CodeOf(err) != "access_token_workspace_management_forbidden" {
		t.Fatalf("flag must be checked before looking up an archived workspace: %v", err)
	}
}
