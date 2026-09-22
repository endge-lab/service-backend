package configurator_auth

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type UseCase interface {
	ResolveCurrentActor(context.Context, ports.UpsertCurrentUserInput, bool, *entities.ExternalAccessSnapshot) (*entities.User, bool, error)
}

func BindUseCase(value *access_control.UseCase) UseCase { return value }
