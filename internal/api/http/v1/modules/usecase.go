package modules

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	catalog "github.com/endge-lab/service-backend/internal/usecase/modules"
)

const maxArchiveSize = catalog.MaxArchiveSize

type UseCase interface {
	AuthorizePublish(context.Context) error
	List(context.Context) ([]entities.Module, error)
	Detail(context.Context, string) (entities.ModuleDetail, error)
	Create(context.Context, entities.Module, []byte) (entities.ModuleDetail, error)
	Patch(context.Context, string, int64, *string, *string) (entities.Module, error)
	Publish(context.Context, string, []byte) (entities.ModuleVersion, error)
	Version(context.Context, string, string, bool) (entities.ModuleVersion, error)
	ReadAsset([]byte, string) ([]byte, error)
}

func BindUseCase(useCase *catalog.UseCase) UseCase { return useCase }
