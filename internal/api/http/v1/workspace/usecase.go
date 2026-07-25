package http

import (
	"context"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
)

type UseCase interface {
	Create(context.Context, workspaces.CreateWorkspaceInput) (*entities.RWorkspace, error)
	List(context.Context) ([]*entities.RWorkspace, error)
	GetByIdentity(context.Context, string) (*entities.RWorkspace, error)
	Update(context.Context, workspaces.UpdateWorkspaceInput) (*entities.RWorkspace, error)
}
