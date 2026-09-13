package build_profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

type Patch struct {
	DisplayName *string
	Visibility  *string
	Settings    *entities.BuildProfileSettings
}

type UseCase struct {
	repository ports.BuildProfileRepository
	tx         ports.TxManager
}

func NewUseCase(repository ports.BuildProfileRepository, tx ports.TxManager) *UseCase {
	return &UseCase{repository: repository, tx: tx}
}

func (u *UseCase) List(ctx context.Context) ([]entities.BuildProfile, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	values, err := u.repository.ListBuildProfiles(ctx, scope.Workspace.ID, actor.User.ID)
	if err != nil {
		return nil, err
	}
	for index := range values {
		values[index] = expose(values[index], actor.User.ID, shared.CanWrite(scope.Role))
	}
	return values, nil
}

func (u *UseCase) Create(ctx context.Context, visibility string, settings entities.BuildProfileSettings) (*entities.BuildProfile, error) {
	actor, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	visibility, err = validateVisibility(visibility)
	if err != nil {
		return nil, err
	}
	settingsJSON, err := ValidateSettings(settings)
	if err != nil {
		return nil, err
	}
	var created *entities.BuildProfile
	err = u.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if lockErr := u.repository.LockBuildProfileNames(txctx, scope.Workspace.ID); lockErr != nil {
			return lockErr
		}
		number, nextErr := u.repository.NextBuildProfileNumber(txctx, scope.Workspace.ID)
		if nextErr != nil {
			return nextErr
		}
		created, nextErr = u.repository.InsertBuildProfile(txctx, entities.BuildProfile{
			ID: uuid.NewString(), Identity: uuid.NewString(), WorkspaceID: scope.Workspace.ID,
			DisplayName: "Новый профиль " + strconv.Itoa(number), Visibility: visibility,
			OwnerUserID: actor.User.ID, SettingsVersion: entities.BuildProfileSettingsVersion,
			Settings: settingsJSON, CreatedBy: entities.Actor{ID: actor.User.ID},
		})
		return nextErr
	})
	if err != nil {
		return nil, shared.MapConflict(err)
	}
	value := expose(*created, actor.User.ID, true)
	return &value, nil
}

func (u *UseCase) Patch(ctx context.Context, identity string, patch Patch, expected int) (*entities.BuildProfile, error) {
	actor, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = uuid.Parse(identity); err != nil {
		return nil, domainerrors.InvalidInput("build_profile.identity_invalid", "profile identity must be UUID")
	}
	if patch.DisplayName == nil && patch.Visibility == nil && patch.Settings == nil {
		return nil, domainerrors.InvalidInput("build_profile.patch_empty", "at least one field is required")
	}
	current, err := u.repository.GetBuildProfile(ctx, scope.Workspace.ID, identity)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if !canManage(*current, actor.User.ID, true) {
		return nil, domainerrors.Forbidden("build_profile_forbidden", "Build profile cannot be managed by this user")
	}
	next := *current
	if patch.DisplayName != nil {
		next.DisplayName, err = validateDisplayName(*patch.DisplayName)
		if err != nil {
			return nil, err
		}
	}
	if patch.Visibility != nil {
		next.Visibility, err = validateVisibility(*patch.Visibility)
		if err != nil {
			return nil, err
		}
		if next.Visibility != current.Visibility && current.OwnerUserID != actor.User.ID {
			return nil, domainerrors.Forbidden("build_profile_visibility_forbidden", "Only the profile owner can change visibility")
		}
	}
	if patch.Settings != nil {
		next.Settings, err = ValidateSettings(*patch.Settings)
		if err != nil {
			return nil, err
		}
		next.SettingsVersion = entities.BuildProfileSettingsVersion
	}
	next.UpdatedBy = entities.Actor{ID: actor.User.ID}
	updated, err := u.repository.UpdateBuildProfile(ctx, next, expected)
	if err != nil {
		return nil, shared.MapConflict(err)
	}
	value := expose(*updated, actor.User.ID, true)
	return &value, nil
}

func (u *UseCase) Delete(ctx context.Context, identity string, expected int) error {
	actor, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return err
	}
	if _, err = uuid.Parse(identity); err != nil {
		return domainerrors.InvalidInput("build_profile.identity_invalid", "profile identity must be UUID")
	}
	current, err := u.repository.GetBuildProfile(ctx, scope.Workspace.ID, identity)
	if err != nil {
		return shared.MapNotFound(err)
	}
	if !canManage(*current, actor.User.ID, true) {
		return domainerrors.Forbidden("build_profile_forbidden", "Build profile cannot be managed by this user")
	}
	return shared.MapConflict(u.repository.DeleteBuildProfile(ctx, scope.Workspace.ID, identity, expected))
}

func validateDisplayName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 160 {
		return "", domainerrors.InvalidInput("build_profile.display_name_invalid", "displayName is required and must not exceed 160 characters")
	}
	return value, nil
}

func validateVisibility(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != entities.BuildProfileVisibilityShared && value != entities.BuildProfileVisibilityPrivate {
		return "", domainerrors.InvalidInput("build_profile.visibility_invalid", "visibility must be shared or private")
	}
	return value, nil
}

// ValidateSettings normalizes the versioned build-settings contract shared by CRUD and portability.
func ValidateSettings(value entities.BuildProfileSettings) (json.RawMessage, error) {
	if value.BuildScope != "complete-model" || value.Contexts != "all-contexts" {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "buildScope and contexts are not supported")
	}
	if value.Diagnostics != "minimal" && value.Diagnostics != "standard" && value.Diagnostics != "detailed" {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "diagnostics is not supported")
	}
	if value.DebuggerStructure != "complete-catalog" && value.DebuggerStructure != "extended-catalog" {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "debuggerStructure is not supported")
	}
	if len(value.Topology) != 1 || value.Topology[0].Node != "frontend" || value.Topology[0].Runtime != "ts-browser" {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "topology must contain frontend / ts-browser")
	}
	raw, err := json.Marshal(value)
	return raw, err
}

// DecodeSettings rejects fields outside the declared settings version before normalization.
func DecodeSettings(raw json.RawMessage) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value entities.BuildProfileSettings
	if err := decoder.Decode(&value); err != nil {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "settings must match version 1")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, domainerrors.InvalidInput("build_profile.settings_invalid", "settings must contain one JSON object")
	}
	return ValidateSettings(value)
}

func expose(value entities.BuildProfile, actorID string, writer bool) entities.BuildProfile {
	value.OwnedByMe = value.OwnerUserID == actorID
	value.CanManage = canManage(value, actorID, writer)
	value.CanChangeVisibility = writer && value.OwnedByMe
	return value
}

func canManage(value entities.BuildProfile, actorID string, writer bool) bool {
	return writer && (value.Visibility == entities.BuildProfileVisibilityShared || value.OwnerUserID == actorID)
}
