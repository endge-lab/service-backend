package session

import (
	"context"

	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/session"
)

type UseCase interface {
	Current(context.Context) (*resourceusecase.Result, error)
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
