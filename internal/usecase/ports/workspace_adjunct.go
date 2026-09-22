package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type AITransferConnection struct {
	Connection entities.AIProviderConnection
	OwnerLogin string
	Credential []byte
	Models     []entities.AIModelProfile
}

type WorkspaceAdjunctRepository interface {
	ListBuildProfilesForTransfer(context.Context, string, string, string) ([]entities.BuildProfile, error)
	ListAIConnectionsForTransfer(context.Context, string, string, bool) ([]AITransferConnection, error)
	FindUserIDsByNormalizedLogin(context.Context, string) ([]string, error)
	UpsertBuildProfileForTransfer(context.Context, entities.BuildProfile) error
	FindAIConnectionIDForTransfer(context.Context, string, string, string, string) (string, error)
	UpsertAIConnectionForTransfer(context.Context, entities.AIProviderConnection, []byte) (string, error)
	UpsertAIModelForTransfer(context.Context, entities.AIModelProfile) error
}
