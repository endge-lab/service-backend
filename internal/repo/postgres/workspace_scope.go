package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"

	"github.com/google/uuid"
)

// workspaceIDFromContext obtains the scope resolved by the HTTP/request layer.
// Repositories never accept a workspace identity from a persisted document body.
func workspaceIDFromContext(ctx context.Context) (uuid.UUID, error) {
	workspaceID, ok := entities.WorkspaceIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperrors.InvalidInput("workspace_required", "workspace scope is required")
	}

	return workspaceID, nil
}

// requireEntityWorkspace prevents an entity assembled by an application caller
// from being persisted under a workspace different from the request scope.
func requireEntityWorkspace(ctx context.Context, entityWorkspaceID uuid.UUID) (uuid.UUID, error) {
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if entityWorkspaceID != workspaceID {
		return uuid.Nil, apperrors.InvalidInput("workspace_scope_mismatch", "entity workspace does not match request scope")
	}
	return workspaceID, nil
}
