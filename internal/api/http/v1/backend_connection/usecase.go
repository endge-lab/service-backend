package backend_connection

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/backend_connections"
)

type UseCase interface {
	List(context.Context) (resourceusecase.ListResult, error)
	Create(context.Context, string) (*entities.BackendConnection, error)
	Delete(context.Context, string) error
}

// BindUseCase предоставляет application use case HTTP-слою.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
