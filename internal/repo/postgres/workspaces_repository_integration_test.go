//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestWorkspacesRepositoryRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	repository := NewWorkspacesRepository(sqlc.New(pool), observability.NewCore(otel.Tracer("workspace-integration-test"), zap.NewNop()), nil)
	identity := "workspace-" + uuid.NewString()
	created, err := repository.Create(ctx, &entities.RWorkspace{Identity: identity, DisplayName: "Default", Configuration: entities.DefaultEndgeConfiguration()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, cleanupErr := pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id = $1", created.ID)
		if cleanupErr != nil {
			t.Errorf("cleanup workspace: %v", cleanupErr)
		}
	})

	fetched, err := repository.GetByIdentity(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.ID != created.ID || fetched.Configuration.DefaultLocale != "ru" || len(fetched.Configuration.Locales) != 2 {
		t.Fatalf("unexpected fetched workspace: %+v", fetched)
	}

	fetched.DisplayName = "Updated"
	fetched.Configuration.Vars = []entities.EndgeVariable{{Name: "API_URL"}}
	updated, err := repository.Update(ctx, fetched)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "Updated" || len(updated.Configuration.Vars) != 1 {
		t.Fatalf("unexpected updated workspace: %+v", updated)
	}

	items, err := repository.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("list: count=%d err=%v", len(items), err)
	}

	_, err = repository.Create(ctx, &entities.RWorkspace{Identity: identity, DisplayName: "Duplicate", Configuration: entities.DefaultEndgeConfiguration()})
	if errors.CodeOf(err) != "identity_conflict" {
		t.Fatalf("duplicate error code = %q", errors.CodeOf(err))
	}
}

func TestProjectsRepositoryIsolatesWorkspaceScope(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	queries := sqlc.New(pool)
	core := observability.NewCore(otel.Tracer("workspace-scope-integration-test"), zap.NewNop())
	workspaces := NewWorkspacesRepository(queries, core, nil)
	projects := NewProjectsRepository(queries, core, nil)

	workspaceOne, err := workspaces.Create(ctx, &entities.RWorkspace{
		Identity:      "workspace-a-" + uuid.NewString(),
		DisplayName:   "Workspace A",
		Configuration: entities.DefaultEndgeConfiguration(),
	})
	if err != nil {
		t.Fatal(err)
	}
	workspaceTwo, err := workspaces.Create(ctx, &entities.RWorkspace{
		Identity:      "workspace-b-" + uuid.NewString(),
		DisplayName:   "Workspace B",
		Configuration: entities.DefaultEndgeConfiguration(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, workspaceID := range []uuid.UUID{workspaceOne.ID, workspaceTwo.ID} {
			if _, cleanupErr := pool.Exec(cleanupCtx, "DELETE FROM projects WHERE workspace_id = $1", workspaceID); cleanupErr != nil {
				t.Errorf("cleanup projects: %v", cleanupErr)
			}
			if _, cleanupErr := pool.Exec(cleanupCtx, "DELETE FROM workspaces WHERE id = $1", workspaceID); cleanupErr != nil {
				t.Errorf("cleanup workspace: %v", cleanupErr)
			}
		}
	})

	workspaceOneCtx := entities.WithWorkspace(ctx, entities.WorkspaceScope{ID: workspaceOne.ID, Identity: workspaceOne.Identity})
	workspaceTwoCtx := entities.WithWorkspace(ctx, entities.WorkspaceScope{ID: workspaceTwo.ID, Identity: workspaceTwo.Identity})
	created, err := projects.Create(workspaceOneCtx, &entities.RProject{
		WorkspaceID: workspaceOne.ID,
		Identity:    "shared-identity",
		DisplayName: "Scoped project",
		Active:      true,
		Meta:        map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = projects.GetByID(workspaceTwoCtx, created.ID); errors.CodeOf(err) != "not_found" {
		t.Fatalf("cross-workspace get error code = %q, want not_found", errors.CodeOf(err))
	}
	items, err := projects.List(workspaceTwoCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cross-workspace list count = %d, want 0", len(items))
	}

	created.DisplayName = "Attempted cross-workspace update"
	if _, err = projects.Update(workspaceTwoCtx, created); errors.CodeOf(err) != "workspace_scope_mismatch" {
		t.Fatalf("cross-workspace update error code = %q, want workspace_scope_mismatch", errors.CodeOf(err))
	}
}

func TestTenantsRepositoryIsolatesWorkspaceAndResolvesRootFolder(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	queries := sqlc.New(pool)
	core := observability.NewCore(otel.Tracer("tenants-integration-test"), zap.NewNop())
	tenants := NewTenantsRepository(queries, core, nil)

	workspaceOne := createTenantTestWorkspace(t, ctx, queries, "tenant-workspace-a-")
	workspaceTwo := createTenantTestWorkspace(t, ctx, queries, "tenant-workspace-b-")
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, workspaceID := range []uuid.UUID{workspaceOne.ID, workspaceTwo.ID} {
			if _, cleanupErr := pool.Exec(cleanupCtx, "DELETE FROM tenants WHERE workspace_id = $1", workspaceID); cleanupErr != nil {
				t.Errorf("cleanup tenants: %v", cleanupErr)
			}
			if _, cleanupErr := pool.Exec(cleanupCtx, "DELETE FROM folders WHERE workspace_id = $1", workspaceID); cleanupErr != nil {
				t.Errorf("cleanup folders: %v", cleanupErr)
			}
			if _, cleanupErr := pool.Exec(cleanupCtx, "DELETE FROM workspaces WHERE id = $1", workspaceID); cleanupErr != nil {
				t.Errorf("cleanup workspace: %v", cleanupErr)
			}
		}
	})

	workspaceOneCtx := entities.WithWorkspace(ctx, entities.WorkspaceScope{ID: workspaceOne.ID, Identity: workspaceOne.Identity})
	workspaceTwoCtx := entities.WithWorkspace(ctx, entities.WorkspaceScope{ID: workspaceTwo.ID, Identity: workspaceTwo.Identity})

	createdOne, err := tenants.Create(workspaceOneCtx, &entities.RTenant{
		WorkspaceID:   workspaceOne.ID,
		Identity:      "shared-tenant",
		DisplayName:   "Tenant One",
		Code:          "SHARED",
		Configuration: entities.DefaultEndgeConfigurationContribution(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if createdOne.FolderID == nil {
		t.Fatal("tenant created without folder must use root-tenants")
	}

	createdTwo, err := tenants.Create(workspaceTwoCtx, &entities.RTenant{
		WorkspaceID:   workspaceTwo.ID,
		Identity:      "shared-tenant",
		DisplayName:   "Tenant Two",
		Code:          "SHARED",
		Configuration: entities.DefaultEndgeConfigurationContribution(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if createdTwo.FolderID == nil || *createdTwo.FolderID == *createdOne.FolderID {
		t.Fatalf("tenant root folders must be resolved in their own workspace: one=%v two=%v", createdOne.FolderID, createdTwo.FolderID)
	}

	items, err := tenants.List(workspaceTwoCtx, ports.TenantsFilter{FolderID: createdTwo.FolderID})
	if err != nil || len(items) != 1 || items[0].ID != createdTwo.ID {
		t.Fatalf("workspace-two list = %+v, err = %v", items, err)
	}
	if _, err = tenants.GetByIdentity(workspaceTwoCtx, "missing-in-workspace-two"); errors.CodeOf(err) != "not_found" {
		t.Fatalf("cross-workspace get error code = %q, want not_found", errors.CodeOf(err))
	}

	_, err = tenants.Create(workspaceOneCtx, &entities.RTenant{
		WorkspaceID:   workspaceOne.ID,
		Identity:      "another-tenant",
		DisplayName:   "Duplicate code",
		Code:          "SHARED",
		Configuration: entities.DefaultEndgeConfigurationContribution(),
	})
	if errors.CodeOf(err) != "tenant_code_conflict" {
		t.Fatalf("duplicate code error code = %q, want tenant_code_conflict", errors.CodeOf(err))
	}

	createdOne.DisplayName = "Cross-workspace update"
	if _, err = tenants.Update(workspaceTwoCtx, createdOne); errors.CodeOf(err) != "workspace_scope_mismatch" {
		t.Fatalf("cross-workspace update error code = %q, want workspace_scope_mismatch", errors.CodeOf(err))
	}

	if err = tenants.HardDelete(workspaceTwoCtx, createdTwo.Identity); err != nil {
		t.Fatalf("delete workspace-two tenant: %v", err)
	}
	if _, err = tenants.GetByIdentity(workspaceTwoCtx, createdTwo.Identity); errors.CodeOf(err) != "not_found" {
		t.Fatalf("deleted tenant get error code = %q, want not_found", errors.CodeOf(err))
	}
}

func createTenantTestWorkspace(t *testing.T, ctx context.Context, queries *sqlc.Queries, prefix string) *entities.RWorkspace {
	t.Helper()

	identity := prefix + uuid.NewString()
	configuration, err := json.Marshal(entities.DefaultEndgeConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	value, err := queries.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		Identity:      identity,
		DisplayName:   "Tenant integration workspace",
		Configuration: configuration,
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace := &entities.RWorkspace{ID: value.ID, Identity: value.Identity, DisplayName: value.DisplayName}
	_, err = queries.CreateFolder(ctx, sqlc.CreateFolderParams{
		WorkspaceID: workspace.ID,
		ProjectID:   pgtype.UUID{},
		EntityType:  string(entities.FolderEntityTypeTenants),
		Identity:    "root-tenants",
		DisplayName: "root-tenants",
		IsRoot:      true,
		IsSystem:    true,
		Meta:        []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return workspace
}
