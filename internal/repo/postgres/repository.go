package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var documentTables = map[string]string{
	"projects": "projects", "tenants": "tenants", "environments": "environments", "folders": "folders",
	"types": "types", "queries": "queries", "data-views": "data_views", "compositions": "compositions",
	"stores": "stores", "streams": "streams", "updates": "updates", "mocks": "mocks", "components": "components",
	"actions": "actions", "filters": "filters", "converters": "converters", "computations": "computations",
	"vocabs": "vocabs", "i18n-bundles": "i18n_bundles", "auth-profiles": "auth_profiles", "navigations": "navigations", "styles": "styles", "configurations": "configurations",
}

type queryExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type EndgeRepository struct {
	pool                   *pgxpool.Pool
	workspaceSchemaVersion int
}

func NewEndgeRepository(pool *pgxpool.Pool, workspaceSchemaVersion int) *EndgeRepository {
	return &EndgeRepository{pool: pool, workspaceSchemaVersion: workspaceSchemaVersion}
}

func (r *EndgeRepository) executor(ctx context.Context) queryExecutor {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func actorScan(prefix string) string {
	return fmt.Sprintf("jsonb_build_object('id', %s.id::text, 'username', %s.username, 'displayName', %s.display_name)", prefix, prefix, prefix)
}

func mustJSON(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
func stringValue(value any) string       { text, _ := value.(string); return strings.TrimSpace(text) }
func boolValue(value any) bool           { result, _ := value.(bool); return result }

func repositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrNotFound
	}
	return err
}
