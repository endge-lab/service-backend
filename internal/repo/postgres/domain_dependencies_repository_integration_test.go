//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestDomainDependenciesRepositoryReplaceDeleteAndRollback(t *testing.T) {
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
	core := observability.NewCore(otel.Tracer("domain-dependencies-integration-test"), zap.NewNop())
	workspaces := NewWorkspacesRepository(queries, core, nil)
	workspace, err := workspaces.Create(ctx, &entities.RWorkspace{Identity: "dependency-workspace-" + uuid.NewString(), DisplayName: "Dependencies", Configuration: entities.DefaultEndgeConfiguration()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, cleanupErr := pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id = $1", workspace.ID); cleanupErr != nil {
			t.Errorf("cleanup workspace: %v", cleanupErr)
		}
	})

	workspaceCtx := entities.WithWorkspaceID(ctx, workspace.ID)
	repository := NewDomainDependenciesRepository(queries, core, nil)
	owner := entities.DomainDependencyOwner{Type: "type", ID: uuid.New(), Identity: "OrderList"}
	if err = repository.ReplaceForOwner(workspaceCtx, owner, []entities.DomainDependencyReference{{Type: "type", Identity: "Money", SourcePath: "schema.fields[0].type"}}, entities.DomainDependencyVerificationStateVerified, nil); err != nil {
		t.Fatal(err)
	}
	assertDependencyUsageTotal(t, repository, workspaceCtx, "type", "Money", 1)

	if err = repository.ReplaceForOwner(workspaceCtx, owner, []entities.DomainDependencyReference{{Type: "type", Identity: "Number", SourcePath: "schema.fields[0].type"}}, entities.DomainDependencyVerificationStateVerified, nil); err != nil {
		t.Fatal(err)
	}
	assertDependencyUsageTotal(t, repository, workspaceCtx, "type", "Money", 0)
	assertDependencyUsageTotal(t, repository, workspaceCtx, "type", "Number", 1)

	tx := NewTxManager(pool, core, nil)
	rollbackOwner := entities.DomainDependencyOwner{Type: "filter", ID: uuid.New(), Identity: "rollback"}
	rollbackErr := errors.New("rollback dependency projection")
	err = tx.WithinTransaction(workspaceCtx, func(txCtx context.Context) error {
		if replaceErr := repository.ReplaceForOwner(txCtx, rollbackOwner, []entities.DomainDependencyReference{{Type: "type", Identity: "RollbackType", SourcePath: "fields[0].type"}}, entities.DomainDependencyVerificationStateUnverified, nil); replaceErr != nil {
			return replaceErr
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("rollback error = %v", err)
	}
	assertDependencyUsageTotal(t, repository, workspaceCtx, "type", "RollbackType", 0)

	if err = repository.DeleteForOwner(workspaceCtx, owner.Type, owner.ID); err != nil {
		t.Fatal(err)
	}
	assertDependencyUsageTotal(t, repository, workspaceCtx, "type", "Number", 0)
}

func assertDependencyUsageTotal(t *testing.T, repository ports.DomainDependenciesRepository, ctx context.Context, dependencyType, dependencyIdentity string, want int64) {
	t.Helper()
	usages, err := repository.ListUsages(ctx, dependencyType, dependencyIdentity, ports.DomainDependenciesListOptions{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if usages.Total != want {
		t.Fatalf("usages for %s/%s = %d, want %d", dependencyType, dependencyIdentity, usages.Total, want)
	}
}
