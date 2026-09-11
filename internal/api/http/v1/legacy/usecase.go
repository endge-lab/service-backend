package legacy

import (
	"context"

	legacyusecase "github.com/endge-lab/service-backend/internal/usecase/legacy"
)

// UseCase задаёт HTTP-контракт временных миграционных операций.
type UseCase interface {
	RebuildWorkspaceFoldersFromFrontend(context.Context, string) (legacyusecase.RebuildWorkspaceFoldersResult, error)
}

// BindUseCase предоставляет concrete Legacy use case как HTTP-порт.
func BindUseCase(useCase *legacyusecase.UseCase) UseCase { return useCase }
