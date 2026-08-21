//go:build integration

package integration_test

import (
	"context"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
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

// TestVocabSourceMigrationBackfillsLegacy проверяет sourceVersion 1 и env-проекцию legacy Vocab.
func TestVocabSourceMigrationBackfillsLegacy(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	ctx := context.Background()
	if err := database.MigrateDownTo(ctx, 51); err != nil {
		t.Fatalf("откатить Vocab source migration: %v", err)
	}

	_, err := database.Pool.Exec(ctx, `INSERT INTO vocabs(
		workspace_id, identity, display_name, data, created_by, updated_by
	) VALUES (
		'00000000-0000-0000-0000-000000000010', 'airlines', 'Airlines',
		'{"mode":"external_payload","baseApiUrl":"{ENDPOINT_VOCABS_SERVICE}","collectionSlug":"airlines","authMode":"inherit"}'::jsonb,
		'00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001'
	)`)
	if err != nil {
		t.Fatalf("создать legacy Vocab: %v", err)
	}
	if err = database.MigrateUp(ctx); err != nil {
		t.Fatalf("применить Vocab source migration: %v", err)
	}

	var source string
	var sourceVersion int
	if err = database.Pool.QueryRow(ctx, `SELECT data->>'source', (data->>'sourceVersion')::int FROM vocabs WHERE identity='airlines'`).Scan(&source, &sourceVersion); err != nil {
		t.Fatalf("прочитать migrated Vocab: %v", err)
	}
	for _, fragment := range []string{
		"defineVocab({", `baseUrl: env("ENDPOINT_VOCABS_SERVICE")`, `collection: "airlines"`, `auth: { mode: "inherit" }`, "items: output().from(response())",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("migrated source не содержит %q: %s", fragment, source)
		}
	}
	if sourceVersion != 1 {
		t.Fatalf("sourceVersion=%d, ожидался 1", sourceVersion)
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
	if len(statuses) == 0 {
		t.Fatal("список embedded-миграций пуст")
	}
	for index, status := range statuses {
		if status.Source == nil {
			t.Fatalf("неожиданная migration position %d: %#v", index, status.Source)
		}
		if index == 0 && status.Source.Version != 1 {
			t.Fatalf("первая migration имеет version %d, ожидалась 1", status.Source.Version)
		}
		if index > 0 && status.Source.Version != statuses[index-1].Source.Version+1 {
			t.Fatalf("нарушена последовательность migration: после %d идёт %d", statuses[index-1].Source.Version, status.Source.Version)
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
		{name: "system roots", query: `SELECT count(*) FROM folders WHERE workspace_id='00000000-0000-0000-0000-000000000010' AND is_root AND managed_by='system'`, want: expectedSystemRootCount()},
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

func expectedSystemRootCount() int {
	entityTypes := make(map[string]struct{}, len(documents.Collections))
	for _, collection := range documents.Collections {
		if collection == "folders" || collection == "configurations" {
			continue
		}
		entityTypes[entities.FolderEntityType(collection)] = struct{}{}
	}
	return len(entityTypes)
}
