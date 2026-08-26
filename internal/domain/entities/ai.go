package entities

import "time"

type AIProviderConnection struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Adapter       string    `json:"adapter"`
	BaseURL       string    `json:"baseUrl"`
	HasCredential bool      `json:"hasCredential"`
	Enabled       bool      `json:"enabled"`
	ModelCount    int       `json:"modelCount"`
	CreatedBy     string    `json:"createdBy"`
	UpdatedBy     string    `json:"updatedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type AIModelProfile struct {
	ID              string    `json:"id"`
	ConnectionID    string    `json:"connectionId"`
	ConnectionName  string    `json:"connectionName"`
	Adapter         string    `json:"adapter"`
	ProviderModelID string    `json:"providerModelId"`
	DisplayName     string    `json:"displayName"`
	Enabled         bool      `json:"enabled"`
	Default         bool      `json:"isDefault"`
	CreatedBy       string    `json:"createdBy"`
	UpdatedBy       string    `json:"updatedBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type AIModelSnapshot struct {
	ProfileID       string `json:"profileId"`
	ConnectionID    string `json:"connectionId"`
	Adapter         string `json:"adapter"`
	ProviderModelID string `json:"providerModelId"`
	DisplayName     string `json:"displayName"`
}

type AIConversation struct {
	ID           string          `json:"id"`
	WorkspaceID  string          `json:"workspaceId"`
	Model        AIModelSnapshot `json:"model"`
	Archived     bool            `json:"archived"`
	MessageCount int64           `json:"messageCount"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type AIMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Sequence       int64     `json:"sequence"`
	CreatedAt      time.Time `json:"createdAt"`
}

type AIRunEvent struct {
	Type         string    `json:"type"`
	RunID        string    `json:"runId,omitempty"`
	MessageID    string    `json:"messageId,omitempty"`
	Delta        string    `json:"delta,omitempty"`
	ErrorCode    string    `json:"errorCode,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
