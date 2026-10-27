package revision

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/revisions"
)

type UseCase interface {
	List(context.Context, string, string, ports.RevisionListOptions) (ports.RevisionPage, error)
	Get(context.Context, string, string, string) (*entities.Revision, error)
	Restore(context.Context, string, string, string, int) (*entities.Document, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
