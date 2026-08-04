package revision

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/revisions"
)

type UseCase interface {
	List(context.Context, string, string) ([]entities.Revision, error)
	Get(context.Context, string, string, string) (*entities.Revision, error)
	Restore(context.Context, string, string, string, int) (*entities.Document, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
