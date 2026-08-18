package domain

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/portable"
)

type UseCase interface {
	Live(context.Context) (json.RawMessage, error)
	Status(context.Context) (*entities.DomainStatus, error)
	Export(context.Context) (json.RawMessage, error)
	PlanImport(context.Context, entities.PortableBundle) (*entities.ImportPlan, error)
	Import(context.Context, string, string, string) (*entities.SnapshotImportResult, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
