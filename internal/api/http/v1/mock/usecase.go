package mock

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/mocks"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type UseCase interface {
	List(context.Context, ports.DocumentFilter) ([]entities.Document, error)
	Get(context.Context, string, bool) (*entities.Document, error)
	Create(context.Context, documents.CreateInput) (*entities.Document, error)
	Patch(context.Context, string, documents.PatchInput, int) (*entities.Document, error)
	Delete(context.Context, string, int) (*entities.Document, error)
	Restore(context.Context, string, int) (*entities.Document, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт ресурса.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
