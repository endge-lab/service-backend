package ai_catalog

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/ai_catalog"
)

type UseCase interface {
	Adapters(context.Context) ([]string, error)
	ListConnections(context.Context) ([]entities.AIProviderConnection, error)
	CreateConnection(context.Context, string, string, string, string, string, bool) (*entities.AIProviderConnection, error)
	PatchConnection(context.Context, string, resourceusecase.ConnectionPatch) (*entities.AIProviderConnection, error)
	ReplaceCredential(context.Context, string, string) (*entities.AIProviderConnection, error)
	DeleteConnection(context.Context, string) error
	ListModels(context.Context, bool) ([]entities.AIModelProfile, error)
	CreateModel(context.Context, string, string, string, bool, bool) (*entities.AIModelProfile, error)
	PatchModel(context.Context, string, resourceusecase.ModelPatch) (*entities.AIModelProfile, error)
	DeleteModel(context.Context, string) error
}

func BindUseCase(usecase *resourceusecase.UseCase) UseCase { return usecase }
