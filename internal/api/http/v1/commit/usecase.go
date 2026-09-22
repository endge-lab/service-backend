package commit

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/commits"
)

type UseCase interface {
	Plan(context.Context) (*resourceusecase.Plan, error)
	Create(context.Context, string, string, int64) (*entities.Commit, error)
	List(context.Context) ([]entities.Commit, error)
	Get(context.Context, string) (*entities.Commit, error)
	PlanRestore(context.Context, string) (*entities.ImportPlan, error)
	Restore(context.Context, string, int64) (*entities.Commit, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
