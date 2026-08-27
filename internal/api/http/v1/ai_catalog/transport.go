package ai_catalog

import "github.com/endge-lab/service-backend/internal/domain/entities"

type CreateConnectionRequest struct {
	Name       string `json:"name" validate:"required,max=160"`
	Adapter    string `json:"adapter" validate:"required,oneof=anthropic ollama"`
	BaseURL    string `json:"baseUrl"`
	Credential string `json:"credential"`
	Visibility string `json:"visibility" validate:"omitempty,oneof=public private"`
	Enabled    bool   `json:"enabled"`
}

type InitialModelRequest struct {
	ProviderModelID string `json:"providerModelId" validate:"required,max=160"`
	DisplayName     string `json:"displayName" validate:"required,max=160"`
	Enabled         bool   `json:"enabled"`
	Default         bool   `json:"isDefault"`
}

type CreateConnectionWithModelRequest struct {
	Name       string              `json:"name" validate:"required,max=160"`
	Adapter    string              `json:"adapter" validate:"required,oneof=anthropic ollama"`
	BaseURL    string              `json:"baseUrl"`
	Credential string              `json:"credential"`
	Visibility string              `json:"visibility" validate:"omitempty,oneof=public private"`
	Enabled    bool                `json:"enabled"`
	Model      InitialModelRequest `json:"model" validate:"required"`
}

type PatchConnectionRequest struct {
	Name    *string `json:"name" validate:"omitempty,max=160"`
	BaseURL *string `json:"baseUrl"`
	Enabled *bool   `json:"enabled"`
}

type ReplaceCredentialRequest struct {
	Credential string `json:"credential" validate:"required"`
}

type CreateModelRequest struct {
	ConnectionID    string `json:"connectionId" validate:"required,uuid"`
	ProviderModelID string `json:"providerModelId" validate:"required,max=160"`
	DisplayName     string `json:"displayName" validate:"required,max=160"`
	Enabled         bool   `json:"enabled"`
	Default         bool   `json:"isDefault"`
}

type PatchModelRequest struct {
	ProviderModelID *string `json:"providerModelId" validate:"omitempty,max=160"`
	DisplayName     *string `json:"displayName" validate:"omitempty,max=160"`
	Enabled         *bool   `json:"enabled"`
	Default         *bool   `json:"isDefault"`
}

type AdaptersResponse struct {
	Items []string `json:"items"`
}

type ConnectionResponse = entities.AIProviderConnection
type ModelResponse = entities.AIModelProfile

type ConnectionWithModelResponse struct {
	Connection entities.AIProviderConnection `json:"connection"`
	Model      entities.AIModelProfile       `json:"model"`
}

type ConnectionsResponse struct {
	Items []entities.AIProviderConnection `json:"items"`
	Total int                             `json:"total"`
}

type ModelsResponse struct {
	Items []entities.AIModelProfile `json:"items"`
	Total int                       `json:"total"`
}
