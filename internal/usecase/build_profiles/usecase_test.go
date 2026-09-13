package build_profiles

import (
	"context"
	"errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

type buildProfileRepositoryStub struct {
	profiles []entities.BuildProfile
	next     int
}

func (r *buildProfileRepositoryStub) ListBuildProfiles(_ context.Context, workspaceID, actorID string) ([]entities.BuildProfile, error) {
	result := make([]entities.BuildProfile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		if profile.WorkspaceID == workspaceID && (profile.Visibility == entities.BuildProfileVisibilityShared || profile.OwnerUserID == actorID) {
			result = append(result, profile)
		}
	}
	return result, nil
}

func (r *buildProfileRepositoryStub) GetBuildProfile(_ context.Context, workspaceID, identity string) (*entities.BuildProfile, error) {
	for index := range r.profiles {
		if r.profiles[index].WorkspaceID == workspaceID && r.profiles[index].Identity == identity {
			value := r.profiles[index]
			return &value, nil
		}
	}
	return nil, ports.ErrNotFound
}

func (r *buildProfileRepositoryStub) LockBuildProfileNames(context.Context, string) error { return nil }
func (r *buildProfileRepositoryStub) NextBuildProfileNumber(context.Context, string) (int, error) {
	if r.next == 0 {
		r.next = 1
	}
	value := r.next
	r.next++
	return value, nil
}

func (r *buildProfileRepositoryStub) InsertBuildProfile(_ context.Context, value entities.BuildProfile) (*entities.BuildProfile, error) {
	value.Revision = 1
	r.profiles = append(r.profiles, value)
	return &value, nil
}

func (r *buildProfileRepositoryStub) UpdateBuildProfile(_ context.Context, value entities.BuildProfile, expected int) (*entities.BuildProfile, error) {
	for index := range r.profiles {
		if r.profiles[index].Identity != value.Identity {
			continue
		}
		if r.profiles[index].Revision != expected {
			return nil, errors.New("revision conflict")
		}
		value.Revision = expected + 1
		r.profiles[index] = value
		return &value, nil
	}
	return nil, ports.ErrNotFound
}

func (r *buildProfileRepositoryStub) DeleteBuildProfile(_ context.Context, workspaceID, identity string, expected int) error {
	for index := range r.profiles {
		if r.profiles[index].WorkspaceID != workspaceID || r.profiles[index].Identity != identity {
			continue
		}
		if r.profiles[index].Revision != expected {
			return errors.New("revision conflict")
		}
		r.profiles = append(r.profiles[:index], r.profiles[index+1:]...)
		return nil
	}
	return ports.ErrNotFound
}

type buildProfileTxStub struct{}

func (buildProfileTxStub) WithinTransaction(ctx context.Context, run func(context.Context) error) error {
	return run(ctx)
}

func (buildProfileTxStub) WithinReadTransaction(ctx context.Context, run func(context.Context) error) error {
	return run(ctx)
}

func TestBuildProfileCreateListAndVisibilityPolicy(t *testing.T) {
	repository := &buildProfileRepositoryStub{}
	usecase := NewUseCase(repository, buildProfileTxStub{})
	ownerContext := buildProfileContext("owner", "editor")
	otherContext := buildProfileContext("other", "editor")

	created, err := usecase.Create(ownerContext, "private", validBuildProfileSettings())
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if created.DisplayName != "Новый профиль 1" || created.Revision != 1 || !created.OwnedByMe || !created.CanManage || !created.CanChangeVisibility {
		t.Fatalf("unexpected created profile: %#v", created)
	}
	if _, err = uuid.Parse(created.ID); err != nil {
		t.Fatalf("id is not UUID: %q", created.ID)
	}
	if _, err = uuid.Parse(created.Identity); err != nil || created.Identity == created.ID {
		t.Fatalf("identity must be an independent UUID: id=%q identity=%q", created.ID, created.Identity)
	}

	visibleToOther, err := usecase.List(otherContext)
	if err != nil {
		t.Fatalf("list as other editor: %v", err)
	}
	if len(visibleToOther) != 0 {
		t.Fatalf("private profile leaked to another user: %#v", visibleToOther)
	}

	shared := entities.BuildProfileVisibilityShared
	if _, err = usecase.Patch(otherContext, created.Identity, Patch{Visibility: &shared}, 1); domainerrors.CodeOf(err) != "build_profile_forbidden" {
		t.Fatalf("another editor changed private profile: code=%q err=%v", domainerrors.CodeOf(err), err)
	}
	updated, err := usecase.Patch(ownerContext, created.Identity, Patch{Visibility: &shared}, 1)
	if err != nil {
		t.Fatalf("owner shares profile: %v", err)
	}
	if updated.Visibility != "shared" || updated.Revision != 2 {
		t.Fatalf("share result: %#v", updated)
	}

	name := "  Общий профиль  "
	updated, err = usecase.Patch(otherContext, created.Identity, Patch{DisplayName: &name}, 2)
	if err != nil {
		t.Fatalf("editor updates shared profile: %v", err)
	}
	if updated.DisplayName != "Общий профиль" || updated.CanChangeVisibility {
		t.Fatalf("shared profile projection for non-owner: %#v", updated)
	}
	private := entities.BuildProfileVisibilityPrivate
	if _, err = usecase.Patch(otherContext, created.Identity, Patch{Visibility: &private}, 3); domainerrors.CodeOf(err) != "build_profile_visibility_forbidden" {
		t.Fatalf("non-owner made shared profile private: code=%q err=%v", domainerrors.CodeOf(err), err)
	}
}

func TestBuildProfileViewerCannotMutateAndSettingsAreValidated(t *testing.T) {
	usecase := NewUseCase(&buildProfileRepositoryStub{}, buildProfileTxStub{})
	if _, err := usecase.Create(buildProfileContext("viewer", "viewer"), "private", validBuildProfileSettings()); domainerrors.CodeOf(err) != "workspace_editor_required" {
		t.Fatalf("viewer create code=%q err=%v", domainerrors.CodeOf(err), err)
	}
	settings := validBuildProfileSettings()
	settings.Topology[0].Runtime = "node"
	if _, err := ValidateSettings(settings); domainerrors.CodeOf(err) != "build_profile.settings_invalid" {
		t.Fatalf("invalid settings code=%q err=%v", domainerrors.CodeOf(err), err)
	}
	unknown := []byte(`{"buildScope":"complete-model","contexts":"all-contexts","diagnostics":"detailed","debuggerStructure":"complete-catalog","topology":[{"node":"frontend","runtime":"ts-browser"}],"future":true}`)
	if _, err := DecodeSettings(unknown); domainerrors.CodeOf(err) != "build_profile.settings_invalid" {
		t.Fatalf("unknown settings field code=%q err=%v", domainerrors.CodeOf(err), err)
	}
}

func buildProfileContext(userID, role string) context.Context {
	ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: userID}})
	return entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{
		Workspace: entities.Workspace{ID: "workspace-id", Identity: "workspace"},
		Role:      role,
	})
}

func validBuildProfileSettings() entities.BuildProfileSettings {
	return entities.BuildProfileSettings{
		BuildScope: "complete-model", Contexts: "all-contexts", Diagnostics: "detailed",
		DebuggerStructure: "complete-catalog",
		Topology:          []entities.BuildProfileTopologyNode{{Node: "frontend", Runtime: "ts-browser"}},
	}
}
