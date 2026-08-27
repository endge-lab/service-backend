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

type CreateConnectionWithModelInput struct {
	Name            string
	Adapter         string
	BaseURL         string
	Credential      string
	Visibility      string
	Enabled         bool
	ProviderModelID string
	DisplayName     string
	ModelEnabled    bool
	MakeDefault     bool
}

type CreatedConnectionWithModel struct {
	Connection entities.AIProviderConnection
	Model      entities.AIModelProfile
}

// ProviderAccess is decrypted only while preparing an authorized Workbench
// run. Callers must not persist, log or expose Credential.
type ProviderAccess struct {
	ConnectionID string
	BaseURL      string
	Credential   string
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
	if _, err := shared.Actor(ctx); err != nil {
		return nil, err
	}
	return []string{"anthropic", "ollama"}, nil
}

func (u *UseCase) ListConnections(ctx context.Context) ([]entities.AIProviderConnection, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	items, err := u.repository.ListAIProviderConnections(ctx, actor.User.ID)
	if err != nil {
		return nil, err
	}
	result := make([]entities.AIProviderConnection, 0, len(items))
	for _, item := range items {
		result = append(result, exposeConnection(item, actor))
	}
	return result, nil
}

func (u *UseCase) CreateConnection(ctx context.Context, name, adapter, baseURL, credential, visibility string, enabled bool) (*entities.AIProviderConnection, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	visibility, err = normalizeVisibility(visibility, actor.PlatformAdmin)
	if err != nil {
		return nil, err
	}
	if visibility == entities.AIVisibilityPublic && !actor.PlatformAdmin {
		return nil, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required for public AI connections")
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
	ownerUserID := ""
	if visibility == entities.AIVisibilityPrivate {
		ownerUserID = actor.User.ID
	}
	created, err := u.repository.InsertAIProviderConnection(ctx, entities.AIProviderConnection{
		ID: id, Name: name, Adapter: adapter, BaseURL: baseURL, Visibility: visibility,
		OwnerUserID: ownerUserID, Enabled: enabled, CreatedBy: actor.User.ID,
	}, encrypted)
	if created != nil {
		value := exposeConnection(*created, actor)
		created = &value
	}
	return created, shared.MapConflict(err)
}

func (u *UseCase) CreateConnectionWithModel(ctx context.Context, input CreateConnectionWithModelInput) (*CreatedConnectionWithModel, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	visibility, err := normalizeVisibility(input.Visibility, actor.PlatformAdmin)
	if err != nil {
		return nil, err
	}
	if visibility == entities.AIVisibilityPublic && !actor.PlatformAdmin {
		return nil, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required for public AI connections")
	}
	if input.MakeDefault && visibility == entities.AIVisibilityPrivate {
		return nil, domainerrors.InvalidInput("ai.private_default_forbidden", "Private AI models cannot be the platform default")
	}

	connectionID := uuid.NewString()
	name, err := normalizeName(input.Name, "connection name")
	if err != nil {
		return nil, err
	}
	adapter := strings.ToLower(strings.TrimSpace(input.Adapter))
	baseURL, err := normalizeBaseURL(adapter, input.BaseURL)
	if err != nil {
		return nil, err
	}
	if adapter != "anthropic" && adapter != "ollama" {
		return nil, domainerrors.InvalidInput("ai.adapter_invalid", "adapter must be anthropic or ollama")
	}
	providerModelID, err := normalizeName(input.ProviderModelID, "provider model id")
	if err != nil {
		return nil, err
	}
	displayName, err := normalizeName(input.DisplayName, "display name")
	if err != nil {
		return nil, err
	}

	var encrypted []byte
	if credential := strings.TrimSpace(input.Credential); credential != "" {
		encrypted, err = u.keyring.Encrypt(credential, credentialAAD(connectionID))
		if err != nil {
			return nil, fmt.Errorf("encrypt provider credential: %w", err)
		}
	}
	ownerUserID := ""
	if visibility == entities.AIVisibilityPrivate {
		ownerUserID = actor.User.ID
	}
	connection := entities.AIProviderConnection{
		ID: connectionID, Name: name, Adapter: adapter, BaseURL: baseURL, Visibility: visibility,
		OwnerUserID: ownerUserID, Enabled: input.Enabled, CreatedBy: actor.User.ID,
	}
	model := entities.AIModelProfile{
		ID: uuid.NewString(), ConnectionID: connectionID, ProviderModelID: providerModelID,
		DisplayName: displayName, Enabled: input.ModelEnabled, Default: input.ModelEnabled && input.MakeDefault, CreatedBy: actor.User.ID,
	}

	var createdConnection *entities.AIProviderConnection
	var createdModel *entities.AIModelProfile
	err = u.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		var createErr error
		createdConnection, createErr = u.repository.InsertAIProviderConnection(txctx, connection, encrypted)
		if createErr != nil {
			return shared.MapConflict(createErr)
		}
		if model.Default {
			if clearErr := u.repository.ClearAIModelDefaults(txctx, model.ID); clearErr != nil {
				return clearErr
			}
		}
		createdModel, createErr = u.repository.InsertAIModelProfile(txctx, model)
		if createErr != nil {
			return shared.MapConflict(createErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if createdConnection == nil || createdModel == nil {
		return nil, fmt.Errorf("create AI connection with model: repository returned an empty result")
	}
	createdConnection.ModelCount = 1
	createdConnectionValue := exposeConnection(*createdConnection, actor)
	createdModelValue := exposeModel(*createdModel, actor)
	return &CreatedConnectionWithModel{Connection: createdConnectionValue, Model: createdModelValue}, nil
}

func (u *UseCase) PatchConnection(ctx context.Context, id string, patch ConnectionPatch) (*entities.AIProviderConnection, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	record, err := u.repository.GetAIProviderConnection(ctx, id, actor.User.ID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if !canManageConnection(record.Connection, actor) {
		return nil, domainerrors.Forbidden("ai.connection_forbidden", "AI connection cannot be managed by this user")
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
	if updated != nil {
		result := exposeConnection(*updated, actor)
		updated = &result
	}
	return updated, shared.MapConflict(shared.MapNotFound(err))
}

func (u *UseCase) ReplaceCredential(ctx context.Context, id, credential string) (*entities.AIProviderConnection, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	record, err := u.repository.GetAIProviderConnection(ctx, id, actor.User.ID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if !canManageConnection(record.Connection, actor) {
		return nil, domainerrors.Forbidden("ai.connection_forbidden", "AI connection cannot be managed by this user")
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
	if updated != nil {
		result := exposeConnection(*updated, actor)
		updated = &result
	}
	return updated, shared.MapNotFound(err)
}

func (u *UseCase) DeleteConnection(ctx context.Context, id string) error {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	record, err := u.repository.GetAIProviderConnection(ctx, id, actor.User.ID)
	if err != nil {
		return shared.MapNotFound(err)
	}
	if !canManageConnection(record.Connection, actor) {
		return domainerrors.Forbidden("ai.connection_forbidden", "AI connection cannot be managed by this user")
	}
	return shared.MapNotFound(u.repository.DeleteAIProviderConnection(ctx, id))
}

func (u *UseCase) ListModels(ctx context.Context, enabledOnly bool) ([]entities.AIModelProfile, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	items, err := u.repository.ListAIModelProfiles(ctx, enabledOnly, actor.User.ID)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = exposeModel(items[index], actor)
	}
	return items, nil
}

func (u *UseCase) CreateModel(ctx context.Context, connectionID, providerModelID, displayName string, enabled, makeDefault bool) (*entities.AIModelProfile, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(connectionID); err != nil {
		return nil, domainerrors.NotFound("ai.connection_not_found", "AI provider connection not found")
	}
	connection, err := u.repository.GetAIProviderConnection(ctx, connectionID, actor.User.ID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if !canManageConnection(connection.Connection, actor) {
		return nil, domainerrors.Forbidden("ai.connection_forbidden", "AI connection cannot be managed by this user")
	}
	if makeDefault && connection.Connection.Visibility == entities.AIVisibilityPrivate {
		return nil, domainerrors.InvalidInput("ai.private_default_forbidden", "Private AI models cannot be the platform default")
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
	if created != nil {
		created.Visibility = connection.Connection.Visibility
		created.OwnerUserID = connection.Connection.OwnerUserID
		value := exposeModel(*created, actor)
		created = &value
	}
	return created, err
}

func (u *UseCase) PatchModel(ctx context.Context, id string, patch ModelPatch) (*entities.AIModelProfile, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	value, err := u.repository.GetAIModelProfile(ctx, id, actor.User.ID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if !canManageModel(*value, actor) {
		return nil, domainerrors.Forbidden("ai.model_forbidden", "AI model cannot be managed by this user")
	}
	if patch.Default != nil && *patch.Default && value.Visibility == entities.AIVisibilityPrivate {
		return nil, domainerrors.InvalidInput("ai.private_default_forbidden", "Private AI models cannot be the platform default")
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
	if updated != nil {
		result := exposeModel(*updated, actor)
		updated = &result
	}
	return updated, err
}

func (u *UseCase) DeleteModel(ctx context.Context, id string) error {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return domainerrors.NotFound("ai.model_not_found", "AI model profile not found")
	}
	value, err := u.repository.GetAIModelProfile(ctx, id, actor.User.ID)
	if err != nil {
		return shared.MapNotFound(err)
	}
	if !canManageModel(*value, actor) {
		return domainerrors.Forbidden("ai.model_forbidden", "AI model cannot be managed by this user")
	}
	return shared.MapNotFound(u.repository.DeleteAIModelProfile(ctx, id))
}

func (u *UseCase) ResolveEnabledModel(ctx context.Context, id string) (*entities.AIModelProfile, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	value, err := u.repository.GetAIModelProfile(ctx, id, actor.User.ID)
	if err != nil {
		return nil, domainerrors.Conflict("ai.model_unavailable", "Selected AI model is no longer available")
	}
	connection, err := u.repository.GetAIProviderConnection(ctx, value.ConnectionID, actor.User.ID)
	if err != nil || !value.Enabled || !connection.Connection.Enabled {
		return nil, domainerrors.Conflict("ai.model_unavailable", "Selected AI model is no longer available")
	}
	return value, nil
}

func (u *UseCase) ResolveProviderAccess(ctx context.Context, connectionID string) (ProviderAccess, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return ProviderAccess{}, err
	}
	record, err := u.repository.GetAIProviderConnection(ctx, connectionID, actor.User.ID)
	if err != nil || !record.Connection.Enabled {
		return ProviderAccess{}, domainerrors.Conflict("ai.model_unavailable", "Selected AI model is no longer available")
	}

	credential := ""
	if len(record.Credential) > 0 {
		credential, err = u.keyring.Decrypt(record.Credential, credentialAAD(record.Connection.ID))
		if err != nil {
			return ProviderAccess{}, fmt.Errorf("decrypt AI provider credential: %w", err)
		}
	}
	return ProviderAccess{
		ConnectionID: record.Connection.ID,
		BaseURL:      record.Connection.BaseURL,
		Credential:   credential,
	}, nil
}

func exposeConnection(value entities.AIProviderConnection, actor entities.CurrentActor) entities.AIProviderConnection {
	value.OwnedByMe = value.Visibility == entities.AIVisibilityPrivate && value.OwnerUserID == actor.User.ID
	value.CanManage = canManageConnection(value, actor)
	return value
}

func exposeModel(value entities.AIModelProfile, actor entities.CurrentActor) entities.AIModelProfile {
	value.OwnedByMe = value.Visibility == entities.AIVisibilityPrivate && value.OwnerUserID == actor.User.ID
	value.CanManage = canManageModel(value, actor)
	return value
}

func canManageConnection(value entities.AIProviderConnection, actor entities.CurrentActor) bool {
	if value.Visibility == entities.AIVisibilityPublic {
		return actor.PlatformAdmin
	}
	return value.Visibility == entities.AIVisibilityPrivate && value.OwnerUserID == actor.User.ID
}

func canManageModel(value entities.AIModelProfile, actor entities.CurrentActor) bool {
	if value.Visibility == entities.AIVisibilityPublic {
		return actor.PlatformAdmin
	}
	return value.Visibility == entities.AIVisibilityPrivate && value.OwnerUserID == actor.User.ID
}

func normalizeVisibility(value string, platformAdmin bool) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		if platformAdmin {
			return entities.AIVisibilityPublic, nil
		}
		return entities.AIVisibilityPrivate, nil
	}
	if value != entities.AIVisibilityPublic && value != entities.AIVisibilityPrivate {
		return "", domainerrors.InvalidInput("ai.visibility_invalid", "visibility must be public or private")
	}
	return value, nil
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
