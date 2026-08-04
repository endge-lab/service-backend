package backup

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/backups"
)

type UseCase interface {
	Create(context.Context, *string) (*entities.SnapshotBackup, error)
	List(context.Context, string, int, int) ([]entities.SnapshotBackup, error)
	Archive(context.Context, string) ([]entities.SnapshotBackup, error)
	Get(context.Context, string, bool) (*entities.SnapshotBackup, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
