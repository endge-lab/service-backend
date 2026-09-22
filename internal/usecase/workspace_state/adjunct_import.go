package workspace_state

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/build_profiles"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

func (s *Coordinator) normalizeAdjunctForImport(
	ctx context.Context,
	bundle *entities.PortableBundle,
	current entities.CurrentActor,
	plan *entities.ImportPlan,
) error {
	profiles := make([]entities.PortableBuildProfile, 0, len(bundle.BuildProfiles))
	seenProfiles := map[string]bool{}
	for index, profile := range bundle.BuildProfiles {
		profile.Identity = strings.TrimSpace(profile.Identity)
		profile.DisplayName = strings.TrimSpace(profile.DisplayName)
		profile.Visibility = strings.ToLower(strings.TrimSpace(profile.Visibility))
		profile.OwnerLogin = normalizeOwnerLogin(profile.OwnerLogin)
		field := fmt.Sprintf("buildProfiles[%d]", index)
		if _, err := uuid.Parse(profile.Identity); err != nil {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".identity must be UUID")
			continue
		}
		if seenProfiles[profile.Identity] {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+" duplicates identity "+profile.Identity)
			continue
		}
		seenProfiles[profile.Identity] = true
		if profile.DisplayName == "" || len([]rune(profile.DisplayName)) > 160 {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".displayName is required and must not exceed 160 characters")
			continue
		}
		if profile.Visibility != entities.BuildProfileVisibilityShared && profile.Visibility != entities.BuildProfileVisibilityPrivate {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".visibility must be shared or private")
			continue
		}
		if profile.SettingsVersion != entities.BuildProfileSettingsVersion {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".settingsVersion is not supported")
			continue
		}
		normalizedSettings, err := build_profiles.DecodeSettings(profile.Settings)
		if err != nil {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".settings is invalid")
			continue
		}
		profile.Settings = normalizedSettings
		ownerIDs, err := s.resolveOwner(ctx, profile.OwnerLogin)
		if err != nil {
			return err
		}
		if len(ownerIDs) != 1 {
			if profile.Visibility == entities.BuildProfileVisibilityPrivate {
				plan.Incoming.SkippedBuildProfiles++
				plan.Warnings = append(plan.Warnings, ownerMappingWarning(field, profile.OwnerLogin, "private build profile was skipped", len(ownerIDs)))
				continue
			}
			plan.Warnings = append(plan.Warnings, ownerMappingWarning(field, profile.OwnerLogin, "shared build profile owner was replaced with the importing administrator", len(ownerIDs)))
		}
		profiles = append(profiles, profile)
		plan.Incoming.BuildProfiles++
	}
	bundle.BuildProfiles = profiles

	if bundle.AICatalog == nil {
		return nil
	}
	connections := make([]entities.PortableAIConnection, 0, len(bundle.AICatalog.Connections))
	seenConnections := map[string]bool{}
	for index, connection := range bundle.AICatalog.Connections {
		connection.Name = strings.TrimSpace(connection.Name)
		connection.Adapter = strings.ToLower(strings.TrimSpace(connection.Adapter))
		connection.BaseURL = strings.TrimSpace(connection.BaseURL)
		connection.Visibility = strings.ToLower(strings.TrimSpace(connection.Visibility))
		connection.OwnerLogin = normalizeOwnerLogin(connection.OwnerLogin)
		field := fmt.Sprintf("aiCatalog.connections[%d]", index)
		if connection.Name == "" || len([]rune(connection.Name)) > 160 {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".name is required and must not exceed 160 characters")
			continue
		}
		if connection.Adapter != entities.AIAdapterAnthropic && connection.Adapter != entities.AIAdapterOllama {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".adapter is not supported")
			continue
		}
		if !validPortableAIBaseURL(connection.Adapter, connection.BaseURL) {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".baseUrl is invalid")
			continue
		}
		ownerID := ""
		if connection.Visibility == entities.AIVisibilityPublic {
			if !current.PlatformAdmin {
				return domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required to import public AI connections")
			}
			connection.OwnerLogin = ""
		} else if connection.Visibility == entities.AIVisibilityPrivate {
			ownerIDs, err := s.resolveOwner(ctx, connection.OwnerLogin)
			if err != nil {
				return err
			}
			if len(ownerIDs) != 1 {
				plan.Incoming.SkippedAIConnections++
				plan.Warnings = append(plan.Warnings, ownerMappingWarning(field, connection.OwnerLogin, "private AI connection was skipped", len(ownerIDs)))
				continue
			}
			ownerID = ownerIDs[0]
			if ownerID != current.User.ID && !current.PlatformAdmin {
				return domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required to import another user's private AI connection")
			}
		} else {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+".visibility must be public or private")
			continue
		}
		key := connection.Visibility + "\x00" + ownerID + "\x00" + connection.Adapter + "\x00" + strings.ToLower(connection.Name)
		if seenConnections[key] {
			plan.Valid = false
			plan.ValidationErrors = append(plan.ValidationErrors, field+" duplicates an AI connection merge key")
			continue
		}
		seenConnections[key] = true
		models := make([]entities.PortableAIModel, 0, len(connection.Models))
		seenModels := map[string]bool{}
		for modelIndex, model := range connection.Models {
			model.ProviderModelID = strings.TrimSpace(model.ProviderModelID)
			model.DisplayName = strings.TrimSpace(model.DisplayName)
			modelField := fmt.Sprintf("%s.models[%d]", field, modelIndex)
			if model.ProviderModelID == "" || len([]rune(model.ProviderModelID)) > 160 || model.DisplayName == "" || len([]rune(model.DisplayName)) > 160 {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, modelField+" names are required and must not exceed 160 characters")
				continue
			}
			if seenModels[model.ProviderModelID] {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, modelField+" duplicates providerModelId")
				continue
			}
			if model.Default && connection.Visibility != entities.AIVisibilityPublic {
				plan.Valid = false
				plan.ValidationErrors = append(plan.ValidationErrors, modelField+" private model cannot be the platform default")
				continue
			}
			seenModels[model.ProviderModelID] = true
			models = append(models, model)
			plan.Incoming.AIModels++
		}
		connection.Models = models
		connections = append(connections, connection)
		plan.Incoming.AIConnections++
	}
	bundle.AICatalog.Connections = connections
	return nil
}

func (s *Coordinator) applyAdjunctImport(ctx context.Context, bundle entities.PortableBundle, current entities.CurrentActor, workspaceID string, result *entities.SnapshotImportResult) error {
	for _, profile := range bundle.BuildProfiles {
		ownerIDs, err := s.resolveOwner(ctx, profile.OwnerLogin)
		if err != nil {
			return err
		}
		ownerID := current.User.ID
		if len(ownerIDs) == 1 {
			ownerID = ownerIDs[0]
		} else if profile.Visibility == entities.BuildProfileVisibilityPrivate {
			return domainerrors.Conflict("import_owner_changed", "Private build profile owner mapping changed after import plan")
		}
		if err = s.repository.UpsertBuildProfileForTransfer(ctx, entities.BuildProfile{
			ID: uuid.NewString(), Identity: profile.Identity, WorkspaceID: workspaceID,
			DisplayName: profile.DisplayName, Visibility: profile.Visibility, OwnerUserID: ownerID,
			SettingsVersion: profile.SettingsVersion, Settings: profile.Settings,
			CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID},
		}); err != nil {
			return err
		}
		result.Imported.BuildProfiles++
	}
	if bundle.AICatalog == nil {
		return nil
	}
	for _, connection := range bundle.AICatalog.Connections {
		ownerID := ""
		if connection.Visibility == entities.AIVisibilityPublic {
			if !current.PlatformAdmin {
				return domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required to import public AI connections")
			}
		} else {
			ownerIDs, err := s.resolveOwner(ctx, connection.OwnerLogin)
			if err != nil {
				return err
			}
			if len(ownerIDs) != 1 {
				return domainerrors.Conflict("import_owner_changed", "Private AI connection owner mapping changed after import plan")
			}
			ownerID = ownerIDs[0]
			if ownerID != current.User.ID && !current.PlatformAdmin {
				return domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required to import another user's private AI connection")
			}
		}
		candidateID, err := s.repository.FindAIConnectionIDForTransfer(ctx, connection.Visibility, ownerID, connection.Adapter, connection.Name)
		if errors.Is(err, ports.ErrNotFound) {
			candidateID = uuid.NewString()
			err = nil
		}
		if err != nil {
			return err
		}
		var encrypted []byte
		if connection.Credential != "" {
			encrypted, err = s.keyring.Encrypt(connection.Credential, []byte("ai-provider-credential:"+candidateID))
			if err != nil {
				return fmt.Errorf("encrypt imported AI credential: %w", err)
			}
		}
		connectionID, err := s.repository.UpsertAIConnectionForTransfer(ctx, entities.AIProviderConnection{
			ID: candidateID, Name: connection.Name, Adapter: connection.Adapter, BaseURL: connection.BaseURL,
			Visibility: connection.Visibility, OwnerUserID: ownerID, Enabled: connection.Enabled,
			CreatedBy: current.User.ID, UpdatedBy: current.User.ID,
		}, encrypted)
		if err != nil {
			return err
		}
		result.Imported.AIConnections++
		for _, model := range connection.Models {
			if err = s.repository.UpsertAIModelForTransfer(ctx, entities.AIModelProfile{
				ID: uuid.NewString(), ConnectionID: connectionID, ProviderModelID: model.ProviderModelID,
				DisplayName: model.DisplayName, Enabled: model.Enabled, Default: model.Default,
				CreatedBy: current.User.ID, UpdatedBy: current.User.ID,
			}); err != nil {
				return err
			}
			result.Imported.AIModels++
		}
	}
	return nil
}

func (s *Coordinator) resolveOwner(ctx context.Context, login string) ([]string, error) {
	login = normalizeOwnerLogin(login)
	if login == "" {
		return nil, nil
	}
	return s.repository.FindUserIDsByNormalizedLogin(ctx, login)
}

func normalizeOwnerLogin(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func ownerMappingWarning(field, login, action string, matches int) string {
	reason := "was not found"
	if matches > 1 {
		reason = "is ambiguous"
	}
	return fmt.Sprintf("%s owner login %q %s; %s", field, login, reason, action)
}

func validPortableAIBaseURL(adapter, value string) bool {
	if adapter == entities.AIAdapterOllama && value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}
