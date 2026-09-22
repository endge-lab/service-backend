package access_control

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_control"
)

type UseCase interface {
	SearchUsers(context.Context, string, string, string, int) (resourceusecase.Page[entities.AccessGrantUser], error)
	List(context.Context, resourceusecase.ListInput) (resourceusecase.Page[entities.AccessGrant], error)
	Put(context.Context, resourceusecase.PutInput) (*entities.AccessGrant, error)
	Delete(context.Context, string) error
	Bulk(context.Context, resourceusecase.BulkInput) (resourceusecase.BulkResult, error)
}

func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
