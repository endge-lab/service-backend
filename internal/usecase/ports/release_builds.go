package ports

import (
	"context"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// ReleaseBuildRepository keeps the optional build separate from metadata reads.
type ReleaseBuildRepository interface {
	StoreReleaseBuild(context.Context, string, string, []byte, entities.ReleaseBuildMetadata) error
	GetReleaseBuild(context.Context, string, string) ([]byte, error)
	LockWorkspaceSnapshot(context.Context, string) error
	GetWorkspace(context.Context, string) (*entities.Workspace, error)
}
