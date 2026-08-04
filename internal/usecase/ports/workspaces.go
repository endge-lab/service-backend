package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// WorkspaceRepository задаёт порт хранения рабочих пространств для use case-слоя.
type WorkspaceRepository interface {
	ListWorkspaces(context.Context, string, bool) ([]entities.Workspace, error)
	GetWorkspace(context.Context, string) (*entities.Workspace, error)
	CreateWorkspace(context.Context, entities.Workspace, string) (*entities.Workspace, error)
	UpdateWorkspace(context.Context, string, map[string]any, int, string) (*entities.Workspace, error)
	WorkspaceRole(context.Context, string, string, bool) (string, error)
	ListMemberships(context.Context, string) ([]entities.Membership, error)
	PutMembership(context.Context, string, string, string, string) (*entities.Membership, error)
	DeleteMembership(context.Context, string, string) error
	ReplaceWorkspaceIntegrations(context.Context, string, []map[string]any, string) error
	ListWorkspaceIntegrations(context.Context, string) ([]map[string]any, error)
}
