package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var documentTables = map[string]string{
	entities.CollectionProjects:       "projects",
	entities.CollectionTenants:        "tenants",
	entities.CollectionEnvironments:   "environments",
	entities.CollectionFolders:        "folders",
	entities.CollectionTypes:          "types",
	entities.CollectionQueries:        "queries",
	entities.CollectionDataViews:      "data_views",
	entities.CollectionCompositions:   "compositions",
	entities.CollectionStores:         "stores",
	entities.CollectionStreams:        "streams",
	entities.CollectionSimulations:    "simulations",
	entities.CollectionUpdates:        "updates",
	entities.CollectionMocks:          "mocks",
	entities.CollectionComponents:     "components",
	entities.CollectionActions:        "actions",
	entities.CollectionFilters:        "filters",
	entities.CollectionConverters:     "converters",
	entities.CollectionComputations:   "computations",
	entities.CollectionVocabs:         "vocabs",
	entities.CollectionI18nBundles:    "i18n_bundles",
	entities.CollectionAuthProfiles:   "auth_profiles",
	entities.CollectionNavigations:    "navigations",
	entities.CollectionStyles:         "styles",
	entities.CollectionConfigurations: "configurations",
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
func defaultStringValue(value any, fallback string) string {
	if result := stringValue(value); result != "" {
		return result
	}
	return fallback
}
func nullableStringValue(value any) *string {
	result := stringValue(value)
	if result == "" {
		return nil
	}
	return &result
}
func boolValue(value any) bool { result, _ := value.(bool); return result }

func repositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrNotFound
	}
	return err
}
