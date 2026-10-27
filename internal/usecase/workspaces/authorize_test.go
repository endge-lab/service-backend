package workspaces

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
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
