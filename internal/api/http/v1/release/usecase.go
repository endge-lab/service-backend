package release

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/releases"
)

type UseCase interface {
	Create(context.Context, resourceusecase.CreateInput) (*entities.Release, error)
	List(context.Context) ([]entities.Release, error)
	Get(context.Context, string) (*entities.Release, error)
	PlanRestore(context.Context, string) (*entities.ImportPlan, error)
	Restore(context.Context, string, int64) (*entities.Commit, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
