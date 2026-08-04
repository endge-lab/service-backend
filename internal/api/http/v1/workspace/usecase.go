package workspace

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/workspaces"
)

type UseCase interface {
	Authorize(context.Context, string) (entities.WorkspaceAccess, error)
	List(context.Context) ([]entities.Workspace, error)
	Get(context.Context, string) (*entities.Workspace, error)
	Create(context.Context, resourceusecase.CreateInput) (*entities.Workspace, error)
	Patch(context.Context, string, resourceusecase.PatchInput, int) (*entities.Workspace, error)
	ListMemberships(context.Context, string) ([]entities.Membership, error)
	PutMembership(context.Context, string, string, string) (*entities.Membership, error)
	DeleteMembership(context.Context, string, string) error
}

// BindUseCase предоставляет concrete application use case как HTTP-порт.
func BindUseCase(useCase *resourceusecase.UseCase) UseCase { return useCase }
