package bridge

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	bridgeusecase "github.com/endge-lab/service-backend/internal/usecase/bridge"
)

type UseCase interface {
	Join(context.Context, string, string, entities.BridgePrincipal, entities.BridgeHello) error
	Handle(context.Context, string, entities.BridgeMessage) error
	Check(context.Context, string) bool
	Leave(string)
}

func BindUseCase(value *bridgeusecase.UseCase) UseCase { return value }
