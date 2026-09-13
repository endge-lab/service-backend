package domain

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/portable"
)

type UseCase interface {
	Live(context.Context) (json.RawMessage, error)
	Status(context.Context) (*entities.DomainStatus, error)
	Export(context.Context) (json.RawMessage, error)
	PlanImport(context.Context, entities.PortableBundle) (*entities.ImportPlan, error)
	Import(context.Context, string, string, string) (*entities.SnapshotImportResult, error)
	ListArchive(context.Context, int, int) (documents.ArchivePage, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(portable *resourceusecase.UseCase, lifecycle *documents.Lifecycle) UseCase {
	return &boundUseCase{UseCase: portable, lifecycle: lifecycle}
}

type boundUseCase struct {
	*resourceusecase.UseCase
	lifecycle *documents.Lifecycle
}

func (u *boundUseCase) ListArchive(ctx context.Context, limit, offset int) (documents.ArchivePage, error) {
	return u.lifecycle.ListArchive(ctx, limit, offset)
}
