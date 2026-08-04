package integration

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/integrations"
)

type UseCase interface {
	List(context.Context, bool) ([]entities.Integration, error)
	Get(context.Context, string, bool) (*entities.Integration, error)
	Create(context.Context, resourceusecase.CreateInput) (*entities.Integration, error)
	Patch(context.Context, string, resourceusecase.PatchInput, int) (*entities.Integration, error)
	Delete(context.Context, string, int) (*entities.Integration, error)
	Restore(context.Context, string, int) (*entities.Integration, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
