package access_token

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_tokens"
)

type UseCase interface {
	Create(context.Context, resourceusecase.CreateInput) (*resourceusecase.Created, error)
	List(context.Context, bool) ([]entities.AccessToken, error)
	Revoke(context.Context, string) error
	MaxTTL() time.Duration
}

func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
