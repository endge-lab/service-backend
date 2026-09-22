package build_profile

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/build_profiles"
)

type UseCase interface {
	List(context.Context) ([]entities.BuildProfile, error)
	Create(context.Context, string, entities.BuildProfileSettings) (*entities.BuildProfile, error)
	Patch(context.Context, string, resourceusecase.Patch, int) (*entities.BuildProfile, error)
	Delete(context.Context, string, int) error
}

func BindUseCase(usecase *resourceusecase.UseCase) UseCase { return usecase }
