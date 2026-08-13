package access_control

import (
	"context"
	"errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type accessRepositoryStub struct {
	ports.AccessControlRepository
	hasAdmins    bool
	platform     map[string]bool
	humanUsers   int
	grants       map[string]entities.AccessGrant
	upsertInputs []ports.AccessGrantInput
}

func (r *accessRepositoryStub) LockBootstrap(context.Context) error { return nil }
func (r *accessRepositoryStub) HasPlatformAdmins(context.Context) (bool, error) {
	return r.hasAdmins, nil
}
func (r *accessRepositoryStub) IsPlatformAdmin(_ context.Context, id string) (bool, error) {
	return r.platform[id], nil
}
func (r *accessRepositoryStub) CountHumanUsers(context.Context) (int, error) {
	return r.humanUsers, nil
}
func (r *accessRepositoryStub) UpsertAccessGrant(_ context.Context, input ports.AccessGrantInput) (*entities.AccessGrant, bool, error) {
	r.upsertInputs = append(r.upsertInputs, input)
	r.platform[input.UserID] = input.ScopeType == "platform"
	if input.ScopeType == "platform" {
		r.hasAdmins = true
	}
	grant := &entities.AccessGrant{ID: "00000000-0000-0000-0000-000000000099", ScopeType: input.ScopeType, Role: input.Role, User: entities.AccessGrantUser{ID: input.UserID, Active: true}}
	r.grants[grant.ID] = *grant
	return grant, true, nil
}
func (r *accessRepositoryStub) GetAccessGrant(_ context.Context, id string) (*entities.AccessGrant, error) {
	grant, ok := r.grants[id]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return &grant, nil
}
func (r *accessRepositoryStub) CountPlatformAdmins(context.Context) (int, error) {
	count := 0
	for _, platform := range r.platform {
		if platform {
			count++
		}
	}
	return count, nil
}

type userRepositoryStub struct {
	ports.UserRepository
	user *entities.User
}

func (r userRepositoryStub) UpsertCurrentUser(context.Context, ports.UpsertCurrentUserInput) (*entities.User, error) {
	return r.user, nil
}

type workspaceRepositoryStub struct {
	ports.WorkspaceRepository
	workspaces map[string]entities.Workspace
	roles      map[string]string
}

func (r workspaceRepositoryStub) GetWorkspace(_ context.Context, identity string) (*entities.Workspace, error) {
	workspace, ok := r.workspaces[identity]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return &workspace, nil
}

func (r workspaceRepositoryStub) WorkspaceRole(_ context.Context, workspaceID, _ string, platform bool) (string, error) {
	if platform {
		return "platform_admin", nil
	}
	return r.roles[workspaceID], nil
}

type txManagerStub struct{}

func (txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (txManagerStub) WithinReadTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestFirstHumanUserBecomesPlatformAdmin(t *testing.T) {
	user := &entities.User{ID: "00000000-0000-0000-0000-000000000010", Active: true}
	repository := &accessRepositoryStub{platform: map[string]bool{}, humanUsers: 1, grants: map[string]entities.AccessGrant{}}
	usecase := NewUseCase(repository, userRepositoryStub{user: user}, workspaceRepositoryStub{}, txManagerStub{})

	resolved, platform, err := usecase.ResolveCurrentActor(context.Background(), ports.UpsertCurrentUserInput{ProviderID: "dev", Issuer: "urn:endge:dev", Subject: "developer"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != user.ID || !platform {
		t.Fatalf("expected first human user to be platform admin, got user=%v platform=%v", resolved.ID, platform)
	}
	if len(repository.upsertInputs) != 1 || repository.upsertInputs[0].ScopeType != "platform" {
		t.Fatalf("expected one platform grant, got %#v", repository.upsertInputs)
	}
}

func TestLegacyPlatformAdminIsPersisted(t *testing.T) {
	user := &entities.User{ID: "00000000-0000-0000-0000-000000000011", Active: true}
	repository := &accessRepositoryStub{hasAdmins: true, platform: map[string]bool{}, humanUsers: 4, grants: map[string]entities.AccessGrant{}}
	usecase := NewUseCase(repository, userRepositoryStub{user: user}, workspaceRepositoryStub{}, txManagerStub{})

	_, platform, err := usecase.ResolveCurrentActor(context.Background(), ports.UpsertCurrentUserInput{}, true)
	if err != nil || !platform {
		t.Fatalf("expected legacy admin persistence, platform=%v err=%v", platform, err)
	}
}

func TestLastPlatformAdminCannotBeDeleted(t *testing.T) {
	userID := "00000000-0000-0000-0000-000000000012"
	grantID := "00000000-0000-0000-0000-000000000099"
	repository := &accessRepositoryStub{
		hasAdmins:  true,
		platform:   map[string]bool{userID: true},
		humanUsers: 1,
		grants: map[string]entities.AccessGrant{
			grantID: {ID: grantID, ScopeType: "platform", Role: "admin", User: entities.AccessGrantUser{ID: userID, Active: true}},
		},
	}
	usecase := NewUseCase(repository, userRepositoryStub{}, workspaceRepositoryStub{}, txManagerStub{})
	ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: userID}, PlatformAdmin: true})

	err := usecase.Delete(ctx, grantID)
	if domainerrors.CodeOf(err) != "last_platform_admin_required" || !errors.Is(err, domainerrors.ErrConflict) {
		t.Fatalf("expected last_platform_admin_required conflict, got %v", err)
	}
}

func TestWorkspaceAdminCannotGrantPlatformRole(t *testing.T) {
	repository := &accessRepositoryStub{platform: map[string]bool{}, grants: map[string]entities.AccessGrant{}}
	usecase := NewUseCase(repository, userRepositoryStub{}, workspaceRepositoryStub{}, txManagerStub{})
	ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "00000000-0000-0000-0000-000000000013"}})

	_, err := usecase.Put(ctx, PutInput{UserID: "00000000-0000-0000-0000-000000000014", ScopeType: "platform", Role: "admin"})
	if domainerrors.CodeOf(err) != "platform_admin_required" {
		t.Fatalf("expected platform_admin_required, got %v", err)
	}
}

func TestOnlyWorkspaceAdminCanManageWorkspaceGrant(t *testing.T) {
	const (
		actorID     = "00000000-0000-0000-0000-000000000013"
		targetID    = "00000000-0000-0000-0000-000000000014"
		workspaceID = "00000000-0000-0000-0000-000000000015"
	)
	for _, role := range []string{"viewer", "editor", "admin"} {
		t.Run(role, func(t *testing.T) {
			repository := &accessRepositoryStub{platform: map[string]bool{}, grants: map[string]entities.AccessGrant{}}
			workspaces := workspaceRepositoryStub{
				workspaces: map[string]entities.Workspace{"production": {ID: workspaceID, Identity: "production", Active: true}},
				roles:      map[string]string{workspaceID: role},
			}
			usecase := NewUseCase(repository, userRepositoryStub{}, workspaces, txManagerStub{})
			ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: actorID}})

			_, err := usecase.Put(ctx, PutInput{UserID: targetID, ScopeType: "workspace", WorkspaceIdentity: "production", Role: "viewer"})
			if role == "admin" {
				if err != nil {
					t.Fatalf("workspace admin must be allowed: %v", err)
				}
				return
			}
			if domainerrors.CodeOf(err) != "workspace_admin_required" {
				t.Fatalf("%s must be rejected, got %v", role, err)
			}
		})
	}
}
