package postgres

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/google/uuid"
)

func TestWorkspaceIDFromContext(t *testing.T) {
	workspaceID := uuid.New()
	ctx := entities.WithWorkspace(context.Background(), entities.WorkspaceScope{
		ID:       workspaceID,
		Identity: "default",
	})

	actualID, err := workspaceIDFromContext(ctx)
	if err != nil {
		t.Fatalf("workspaceIDFromContext() error = %v", err)
	}
	if actualID != workspaceID {
		t.Fatalf("workspaceIDFromContext() = %s, want %s", actualID, workspaceID)
	}
}

func TestWorkspaceIDFromContextRequiresScope(t *testing.T) {
	_, err := workspaceIDFromContext(context.Background())
	if code := errors.CodeOf(err); code != "workspace_required" {
		t.Fatalf("error code = %q, want workspace_required", code)
	}
}

func TestRequireEntityWorkspaceRejectsDifferentScope(t *testing.T) {
	ctx := entities.WithWorkspaceID(context.Background(), uuid.New())

	_, err := requireEntityWorkspace(ctx, uuid.New())
	if code := errors.CodeOf(err); code != "workspace_scope_mismatch" {
		t.Fatalf("error code = %q, want workspace_scope_mismatch", code)
	}
}
