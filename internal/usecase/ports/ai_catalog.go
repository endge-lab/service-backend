package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type AIProviderConnectionRecord struct {
	Connection entities.AIProviderConnection
	Credential []byte
}

type AICatalogRepository interface {
	ListAIProviderConnections(context.Context) ([]entities.AIProviderConnection, error)
	GetAIProviderConnection(context.Context, string) (*AIProviderConnectionRecord, error)
	InsertAIProviderConnection(context.Context, entities.AIProviderConnection, []byte) (*entities.AIProviderConnection, error)
	UpdateAIProviderConnection(context.Context, entities.AIProviderConnection) (*entities.AIProviderConnection, error)
	UpdateAIProviderCredential(context.Context, string, string, []byte) (*entities.AIProviderConnection, error)
	DeleteAIProviderConnection(context.Context, string) error

	ListAIModelProfiles(context.Context, bool) ([]entities.AIModelProfile, error)
	GetAIModelProfile(context.Context, string) (*entities.AIModelProfile, error)
	InsertAIModelProfile(context.Context, entities.AIModelProfile) (*entities.AIModelProfile, error)
	UpdateAIModelProfile(context.Context, entities.AIModelProfile) (*entities.AIModelProfile, error)
	ClearAIModelDefaults(context.Context, string) error
	DeleteAIModelProfile(context.Context, string) error
}
