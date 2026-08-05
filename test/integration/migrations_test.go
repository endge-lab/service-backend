//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/pressly/goose/v3"
)

// TestMigrationsRoundTrip проверяет clean up, idempotent up, полный down и повторный up.
func TestMigrationsRoundTrip(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	ctx := context.Background()

	if err := database.MigrateUp(ctx); err != nil {
		t.Fatalf("повторный up должен быть идемпотентным: %v", err)
	}
	assertMigrationState(t, database, goose.StateApplied)
	assertBootstrapState(t, database)

	if err := database.MigrateDownToZero(ctx); err != nil {
		t.Fatalf("полный down: %v", err)
	}
	assertMigrationState(t, database, goose.StatePending)
	for _, table := range []string{"service_users", "workspaces", "folders", "document_revisions", "workspace_commits", "releases", "workspace_snapshot_backups", "workspace_snapshot_import_plans", "configurator_auth_sessions"} {
		var exists bool
		if err := database.Pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("проверить таблицу %s после down: %v", table, err)
		}
		if exists {
			t.Fatalf("таблица %s осталась после полного down", table)
		}
	}

	if err := database.MigrateUp(ctx); err != nil {
		t.Fatalf("повторный up после down: %v", err)
	}
	assertMigrationState(t, database, goose.StateApplied)
	assertBootstrapState(t, database)
}

// TestMigrationSchemaGuards проверяет bootstrap и отсутствие исключённых MVP-таблиц.
func TestMigrationSchemaGuards(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	ctx := context.Background()
	assertBootstrapState(t, database)

	for _, table := range []string{"parameters", "pages", "page_templates", "policies", "versions", "components_legacy", "domain_dependencies", "domain_dependency_states"} {
		var exists bool
		if err := database.Pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("проверить исключённую таблицу %s: %v", table, err)
		}
		if exists {
			t.Fatalf("исключённая таблица %s присутствует в MVP schema", table)
		}
	}

	var accessTokenColumn, identityRefreshColumn bool
	if err := database.Pool.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='configurator_auth_sessions' AND column_name='access_token_encrypted'),
		EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='configurator_auth_sessions' AND column_name='identity_refresh_at')`).Scan(&accessTokenColumn, &identityRefreshColumn); err != nil {
		t.Fatalf("проверить auth session schema: %v", err)
	}
	if accessTokenColumn || !identityRefreshColumn {
		t.Fatalf("неверная auth session schema: access_token_encrypted=%t identity_refresh_at=%t", accessTokenColumn, identityRefreshColumn)
	}
}

func assertMigrationState(t *testing.T, database interface {
	MigrationStatus(context.Context) ([]*goose.MigrationStatus, error)
}, expected goose.State) {
	t.Helper()
	statuses, err := database.MigrationStatus(context.Background())
	if err != nil {
		t.Fatalf("получить migration status: %v", err)
	}
	if len(statuses) != 41 {
		t.Fatalf("миграций = %d, ожидалось 41", len(statuses))
	}
	for index, status := range statuses {
		if status.Source == nil || status.Source.Version != int64(index+1) {
			t.Fatalf("неожиданная migration position %d: %#v", index, status.Source)
		}
		if status.State != expected {
			t.Fatalf("migration %d имеет state %s, ожидался %s", status.Source.Version, status.State, expected)
		}
	}
}

func assertBootstrapState(t *testing.T, database *support.TestDatabase) {
	t.Helper()
	ctx := context.Background()
	checks := []struct {
		name  string
		query string
		want  int
	}{
		{name: "system user", query: `SELECT count(*) FROM service_users WHERE id='00000000-0000-0000-0000-000000000001' AND is_system`, want: 1},
		{name: "default workspace", query: `SELECT count(*) FROM workspaces WHERE id='00000000-0000-0000-0000-000000000010' AND identity='default'`, want: 1},
		{name: "system roots", query: `SELECT count(*) FROM folders WHERE workspace_id='00000000-0000-0000-0000-000000000010' AND is_root AND managed_by='system'`, want: 21},
		{name: "initial commit", query: `SELECT count(*) FROM workspace_commits WHERE workspace_id='00000000-0000-0000-0000-000000000010' AND operation='bootstrap'`, want: 1},
	}
	for _, check := range checks {
		var count int
		if err := database.Pool.QueryRow(ctx, check.query).Scan(&count); err != nil {
			t.Fatalf("проверить %s: %v", check.name, err)
		}
		if count != check.want {
			t.Fatalf("%s: count=%d, ожидалось %d", check.name, count, check.want)
		}
	}
}
