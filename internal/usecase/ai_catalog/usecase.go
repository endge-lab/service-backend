package ai_catalog

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	platformencryption "github.com/endge-lab/service-backend/internal/platform/encryption"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

type ConnectionPatch struct {
	Name    *string
	BaseURL *string
	Enabled *bool
}

type ModelPatch struct {
	ProviderModelID *string
	DisplayName     *string
	Enabled         *bool
	Default         *bool
}

type UseCase struct {
	repository ports.AICatalogRepository
	tx         ports.TxManager
	keyring    *platformencryption.Keyring
}

func NewUseCase(repository ports.AICatalogRepository, tx ports.TxManager, keyring *platformencryption.Keyring) *UseCase {
	return &UseCase{repository: repository, tx: tx, keyring: keyring}
}

func (u *UseCase) Adapters(ctx context.Context) ([]string, error) {
	if _, err := platformAdmin(ctx); err != nil {
		return nil, err
	}
	return []string{"anthropic", "ollama"}, nil
}

func (u *UseCase) ListConnections(ctx context.Context) ([]entities.AIProviderConnection, error) {
	if _, err := platformAdmin(ctx); err != nil {
		return nil, err
	}
	return u.repository.ListAIProviderConnections(ctx)
}

func (u *UseCase) CreateConnection(ctx context.Context, name, adapter, baseURL, credential string, enabled bool) (*entities.AIProviderConnection, error) {
	actor, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	name, err = normalizeName(name, "connection name")
	if err != nil {
		return nil, err
	}
	adapter = strings.ToLower(strings.TrimSpace(adapter))
	baseURL, err = normalizeBaseURL(adapter, baseURL)
	if err != nil {
		return nil, err
	}
	if adapter != "anthropic" && adapter != "ollama" {
		return nil, domainerrors.InvalidInput("ai.adapter_invalid", "adapter must be anthropic or ollama")
	}
	var encrypted []byte
	if credential = strings.TrimSpace(credential); credential != "" {
		encrypted, err = u.keyring.Encrypt(credential, credentialAAD(id))
		if err != nil {
			return nil, fmt.Errorf("encrypt provider credential: %w", err)
		}
	}
	created, err := u.repository.InsertAIProviderConnection(ctx, entities.AIProviderConnection{
		ID: id, Name: name, Adapter: adapter, BaseURL: baseURL, Enabled: enabled, CreatedBy: actor.User.ID,
	}, encrypted)
	return created, shared.MapConflict(err)
}

func (u *UseCase) PatchConnection(ctx context.Context, id string, patch ConnectionPatch) (*entities.AIProviderConnection, error) {
	actor, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	record, err := u.repository.GetAIProviderConnection(ctx, id)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	value := record.Connection
	if patch.Name != nil {
		value.Name, err = normalizeName(*patch.Name, "connection name")
		if err != nil {
			return nil, err
		}
	}
	if patch.BaseURL != nil {
		value.BaseURL, err = normalizeBaseURL(value.Adapter, *patch.BaseURL)
		if err != nil {
			return nil, err
		}
	}
	if patch.Enabled != nil {
		value.Enabled = *patch.Enabled
	}
	value.UpdatedBy = actor.User.ID
	updated, err := u.repository.UpdateAIProviderConnection(ctx, value)
	return updated, shared.MapConflict(shared.MapNotFound(err))
}

func (u *UseCase) ReplaceCredential(ctx context.Context, id, credential string) (*entities.AIProviderConnection, error) {
	actor, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return nil, domainerrors.InvalidInput("ai.credential_required", "credential is required")
	}
	encrypted, err := u.keyring.Encrypt(credential, credentialAAD(id))
	if err != nil {
		return nil, fmt.Errorf("encrypt provider credential: %w", err)
	}
	updated, err := u.repository.UpdateAIProviderCredential(ctx, id, actor.User.ID, encrypted)
	return updated, shared.MapNotFound(err)
}

func (u *UseCase) DeleteConnection(ctx context.Context, id string) error {
	if _, err := platformAdmin(ctx); err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	return shared.MapNotFound(u.repository.DeleteAIProviderConnection(ctx, id))
}

func (u *UseCase) ListModels(ctx context.Context, enabledOnly bool) ([]entities.AIModelProfile, error) {
	if !enabledOnly {
		if _, err := platformAdmin(ctx); err != nil {
			return nil, err
		}
	} else if _, err := shared.Actor(ctx); err != nil {
		return nil, err
	}
	return u.repository.ListAIModelProfiles(ctx, enabledOnly)
}

func (u *UseCase) CreateModel(ctx context.Context, connectionID, providerModelID, displayName string, enabled, makeDefault bool) (*entities.AIModelProfile, error) {
	actor, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(connectionID); err != nil {
		return nil, domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	providerModelID, err = normalizeName(providerModelID, "provider model id")
	if err != nil {
		return nil, err
	}
	displayName, err = normalizeName(displayName, "display name")
	if err != nil {
		return nil, err
	}
	value := entities.AIModelProfile{
		ID: uuid.NewString(), ConnectionID: connectionID, ProviderModelID: providerModelID,
		DisplayName: displayName, Enabled: enabled, Default: enabled && makeDefault, CreatedBy: actor.User.ID,
	}
	var created *entities.AIModelProfile
	err = u.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if value.Default {
			if clearErr := u.repository.ClearAIModelDefaults(txctx, value.ID); clearErr != nil {
				return clearErr
			}
		}
		var createErr error
		created, createErr = u.repository.InsertAIModelProfile(txctx, value)
		if createErr != nil {
			return shared.MapConflict(createErr)
		}
		return nil
	})
	return created, err
}

func (u *UseCase) PatchModel(ctx context.Context, id string, patch ModelPatch) (*entities.AIModelProfile, error) {
	actor, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	value, err := u.repository.GetAIModelProfile(ctx, id)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if patch.ProviderModelID != nil {
		value.ProviderModelID, err = normalizeName(*patch.ProviderModelID, "provider model id")
		if err != nil {
			return nil, err
		}
	}
	if patch.DisplayName != nil {
		value.DisplayName, err = normalizeName(*patch.DisplayName, "display name")
		if err != nil {
			return nil, err
		}
	}
	if patch.Enabled != nil {
		value.Enabled = *patch.Enabled
		if !value.Enabled {
			value.Default = false
		}
	}
	if patch.Default != nil {
		value.Default = *patch.Default && value.Enabled
	}
	value.UpdatedBy = actor.User.ID
	var updated *entities.AIModelProfile
	err = u.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if value.Default {
			if clearErr := u.repository.ClearAIModelDefaults(txctx, value.ID); clearErr != nil {
				return clearErr
			}
		}
		var updateErr error
		updated, updateErr = u.repository.UpdateAIModelProfile(txctx, *value)
		if updateErr != nil {
			return shared.MapConflict(shared.MapNotFound(updateErr))
		}
		return nil
	})
	return updated, err
}

func (u *UseCase) DeleteModel(ctx context.Context, id string) error {
	if _, err := platformAdmin(ctx); err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return domainerrors.NotFound("ai.model_not_found", "AI model profile not found")
	}
	return shared.MapNotFound(u.repository.DeleteAIModelProfile(ctx, id))
}

func (u *UseCase) ResolveEnabledModel(ctx context.Context, id string) (*entities.AIModelProfile, error) {
	value, err := u.repository.GetAIModelProfile(ctx, id)
	if err != nil {
		return nil, domainerrors.Conflict("ai.model_unavailable", "Selected AI model is no longer available")
	}
	connection, err := u.repository.GetAIProviderConnection(ctx, value.ConnectionID)
	if err != nil || !value.Enabled || !connection.Connection.Enabled {
		return nil, domainerrors.Conflict("ai.model_unavailable", "Selected AI model is no longer available")
	}
	return value, nil
}

func platformAdmin(ctx context.Context) (entities.CurrentActor, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return actor, err
	}
	if !actor.PlatformAdmin {
		return actor, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required")
	}
	return actor, nil
}

func normalizeName(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", domainerrors.InvalidInput("ai.field_required", field+" is required")
	}
	if len([]rune(value)) > 160 {
		return "", domainerrors.InvalidInput("ai.field_too_long", field+" must not exceed 160 characters")
	}
	return value, nil
}

func normalizeBaseURL(adapter, value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" && adapter == "anthropic" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", domainerrors.InvalidInput("ai.base_url_invalid", "baseUrl must be an absolute HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", domainerrors.InvalidInput("ai.base_url_invalid", "baseUrl must use http or https")
	}
	return value, nil
}

func credentialAAD(connectionID string) []byte {
	return []byte("ai-provider-credential:" + connectionID)
}
