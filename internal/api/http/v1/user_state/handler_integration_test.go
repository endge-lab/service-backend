package user_state_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/endge-lab/service-backend/internal/api/http/v1/user_state"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	state "github.com/endge-lab/service-backend/internal/usecase/user_state"
	"github.com/endge-lab/service-backend/migrations"
	"github.com/endge-lab/service-kit-go/pkg/migrator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	trace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

func TestUserStateHTTPPostgres(t *testing.T) {
	dsn := os.Getenv("ENDGE_USER_STATE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("set ENDGE_USER_STATE_TEST_DATABASE to an empty disposable PostgreSQL database")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	require.NoError(t, migrator.NewMigrator(db, migrations.FS, zap.NewNop()).Up())
	require.NoError(t, db.Close())

	actorID, adminID := uuid.NewString(), uuid.NewString()
	for _, id := range []string{actorID, adminID} {
		_, err = pool.Exec(ctx, `INSERT INTO service_users(id,provider_id,subject,issuer,username)
			VALUES ($1::uuid,'test',($1::uuid)::text,'test',($1::uuid)::text)`, id)
		require.NoError(t, err)
	}
	configuration := json.RawMessage(`{"userStateDriver":"backend"}`)
	workspace := entities.Workspace{ID: uuid.NewString(), Identity: "settings-fixture", Configuration: configuration}
	otherWorkspace := entities.Workspace{ID: uuid.NewString(), Identity: "other-fixture", Configuration: configuration}
	for _, ws := range []entities.Workspace{workspace, otherWorkspace} {
		_, err = pool.Exec(ctx, `INSERT INTO workspaces(id,identity,display_name,configuration,created_by,updated_by)
			VALUES ($1,$2,$2,$3,$4,$4)`, ws.ID, ws.Identity, ws.Configuration, adminID)
		require.NoError(t, err)
	}
	repository := postgres.NewEndgeRepository(pool, 10)
	metrics, err := postgres.NewRepositoryMetrics(noop.NewMeterProvider().Meter("test"))
	require.NoError(t, err)
	tx := postgres.NewTxManager(pool, observability.NewCore(trace.NewTracerProvider().Tracer("test"), zap.NewNop()), metrics)
	usecase := state.NewUseCase(repository, repository, tx, "test")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		user, role := c.Get("Test-Actor"), c.Get("Test-Role")
		current := workspace
		if c.Get("Test-Other") == "1" {
			current = otherWorkspace
		}
		context := entities.WithCurrentActor(c.UserContext(), entities.CurrentActor{User: &entities.User{ID: user}})
		c.SetUserContext(entities.WithWorkspaceAccess(context, entities.WorkspaceAccess{Workspace: current, Role: role}))
		return c.Next()
	})
	user_state.RegisterRoutes(app, user_state.NewHandler(usecase))

	requestAs := func(t *testing.T, method, path string, body any, user, role string, other bool, status int, out any) {
		t.Helper()
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(method, "/user-state"+path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Test-Actor", user)
		req.Header.Set("Test-Role", role)
		if other {
			req.Header.Set("Test-Other", "1")
		}
		res, err := app.Test(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, status, res.StatusCode)
		if out != nil {
			require.NoError(t, json.NewDecoder(res.Body).Decode(out))
		}
	}
	request := func(method, path string, body any, admin, other bool, status int, out any) {
		t.Helper()
		user, role := actorID, "viewer"
		if admin {
			user, role = adminID, "admin"
		}
		requestAs(t, method, path, body, user, role, other, status, out)
	}
	getSnapshot := func() state.Snapshot {
		var snapshot state.Snapshot
		request("GET", "/", nil, false, false, 200, &snapshot)
		return snapshot
	}
	patch := func(target string, batch state.Batch, admin bool) state.BatchResult {
		var result state.BatchResult
		request("PATCH", target, batch, admin, false, 200, &result)
		return result
	}
	op := func(key, value string, base int64) state.Operation {
		return state.Operation{Key: key, ID: uuid.NewString(), BaseRevision: base, Value: json.RawMessage(value)}
	}
	zoom := "endge.ui.zoom"
	first := getSnapshot()
	batch := state.Batch{ActorID: first.ActorID, Partition: first.Partition, Operations: []state.Operation{op(zoom, "125", 0)}}
	entry := patch("/", batch, false).Results[0].Entry
	require.Equal(t, entry.Revision, patch("/", batch, false).Results[0].Entry.Revision, "duplicate acknowledged mutation")
	require.JSONEq(t, "125", string(getSnapshot().Entries[0].Value))
	request("GET", "/admin/users", nil, false, false, 200, nil)
	request("PATCH", "/", batch, true, false, 403, nil)
	request("PATCH", "/", batch, false, true, 403, nil)
	var page state.UserPage
	request("GET", "/admin/users", nil, true, false, 200, &page)
	require.Len(t, page.Items, 1)
	require.Equal(t, actorID, page.Items[0].ID)
	var otherPage state.UserPage
	request("GET", "/admin/users", nil, true, true, 200, &otherPage)
	require.Empty(t, otherPage.Items)
	otherEnvironment := state.NewUseCase(repository, repository, tx, "another-deployment")
	envSession := entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: workspace, Role: "viewer"})
	envSession = entities.WithCurrentActor(envSession, entities.CurrentActor{User: &entities.User{ID: actorID}})
	isolated, err := otherEnvironment.Read(envSession, nil, "", false, "")
	require.NoError(t, err)
	require.Empty(t, isolated.Entries)
	require.NotEqual(t, first.Partition, isolated.Partition)

	request("POST", "/admin/reset", map[string]string{"partition": isolated.Partition, "confirmation": "reset-all"}, true, false, 403, nil)
	require.Len(t, getSnapshot().Entries, 1)

	adminChange := patch("/admin/users/"+actorID, state.Batch{Partition: first.Partition,
		Operations: []state.Operation{op(zoom, "150", entry.Revision)}}, true)
	require.Equal(t, "saved", adminChange.Results[0].Status)
	offline := state.Batch{ActorID: first.ActorID, Partition: first.Partition,
		Operations: []state.Operation{op(zoom, "100", entry.Revision), op("configurator.date-format", `"robot"`, 0)}}
	merged := patch("/", offline, false)
	require.Equal(t, "conflict", merged.Results[0].Status)
	require.Equal(t, "saved", merged.Results[1].Status)
	require.JSONEq(t, "150", string(merged.Results[0].Entry.Value))

	request("POST", "/admin/reset", map[string]string{"userId": actorID, "partition": first.Partition}, true, false, 400, nil)
	request("POST", "/admin/reset", map[string]string{"userId": actorID, "confirmation": "reset-user", "partition": first.Partition}, true, false, 200, nil)
	afterReset := getSnapshot()
	require.Empty(t, afterReset.Entries)
	require.Greater(t, afterReset.ResetRevision, int64(0))
	require.Equal(t, "reset", patch("/", offline, false).Results[0].Status)
	batch.ResetRevision = afterReset.ResetRevision
	batch.Operations = []state.Operation{op(zoom, "100", 0)}
	require.Equal(t, "saved", patch("/", batch, false).Results[0].Status)
	request("POST", "/admin/reset", map[string]string{"confirmation": "reset-all", "partition": first.Partition}, true, false, 200, nil)
	require.Empty(t, getSnapshot().Entries)

	_, err = repository.InsertFacet(ctx, entities.Document{ID: uuid.NewString(), WorkspaceID: workspace.ID, Identity: "tenant", DisplayName: "Tenant",
		Meta: json.RawMessage("{}"), Data: json.RawMessage("{}"), Active: true, Revision: 1, CreatedBy: entities.Actor{ID: adminID}})
	require.NoError(t, err)
	_, err = repository.InsertFacetDocument(ctx, entities.Document{
		ID: uuid.NewString(), WorkspaceID: workspace.ID, Identity: "north", DisplayName: "North", Meta: json.RawMessage("{}"),
		Data:   json.RawMessage(`{"facetIdentity":"tenant","configuration":{"mode":"inherit","patch":{"userStateDriver":{"op":"set","value":"backend"}}}}`),
		Active: true, Revision: 1, CreatedBy: entities.Actor{ID: adminID},
	})
	require.NoError(t, err)
	query := "?" + url.Values{"facets": {`{"tenant":"north"}`}}.Encode()
	var own state.Snapshot
	request("GET", "/"+query, nil, false, false, 200, &own)
	require.Empty(t, own.Entries)
	request("GET", "/", nil, false, false, 400, nil)
	request("GET", "/?"+url.Values{"facets": {`{"tenant":"unknown"}`}}.Encode(), nil, false, false, 400, nil)
	ownChange := state.Batch{ActorID: own.ActorID, Partition: own.Partition,
		Operations: []state.Operation{op(zoom, "125", 0)}}
	require.Equal(t, "saved", patch("/"+query, ownChange, false).Results[0].Status)

	target := "/admin/users/" + actorID + query
	for _, role := range []string{"viewer", "editor", "admin", "platform_admin"} {
		t.Run(role, func(t *testing.T) {
			var contexts struct {
				Current *state.Scope
			}
			requestAs(t, "GET", "/admin/contexts"+query, nil, adminID, role, false, 200, &contexts)
			require.NotNil(t, contexts.Current)
			require.Equal(t, "tenant", contexts.Current.Source)
			require.Equal(t, "backend", contexts.Current.Driver)
			var users state.UserPage
			requestAs(t, "GET", "/admin/users"+query, nil, adminID, role, false, 200, &users)
			require.Len(t, users.Items, 1)
			require.Equal(t, actorID, users.Items[0].ID)
			var snapshot state.Snapshot
			requestAs(t, "GET", target, nil, adminID, role, false, 200, &snapshot)
			require.Len(t, snapshot.Entries, 1)

			change := state.Batch{Partition: snapshot.Partition, ResetRevision: snapshot.ResetRevision,
				Operations: []state.Operation{op(zoom, "130", snapshot.Entries[0].Revision)}}
			if role == "viewer" {
				requestAs(t, "PATCH", target, change, adminID, role, false, 403, nil)
			} else {
				var result state.BatchResult
				requestAs(t, "PATCH", target, change, adminID, role, false, 200, &result)
				require.Equal(t, "saved", result.Results[0].Status)
			}
			if role == "viewer" || role == "editor" {
				requestAs(t, "POST", "/admin/reset"+query, map[string]string{
					"partition": snapshot.Partition, "confirmation": "reset-all",
				}, adminID, role, false, 403, nil)
			}
		})
	}

	t.Run("application settings and arbitrary workspace keys", func(t *testing.T) {
		appQuery := query + "&application=fixture-app"
		var personal state.Snapshot
		requestAs(t, "GET", "/"+appQuery, nil, actorID, "viewer", false, 200, &personal)
		require.Equal(t, "fixture-app", personal.Application)
		require.NotEqual(t, own.Partition, personal.Partition)
		change := state.Batch{ActorID: personal.ActorID, Partition: personal.Partition,
			Operations: []state.Operation{op("appearance.theme", `"dark"`, 0), op("aodb.shortcuts", `["KeyA"]`, 0)}}
		var saved state.BatchResult
		requestAs(t, "PATCH", "/"+appQuery, change, actorID, "viewer", false, 200, &saved)
		require.Equal(t, "saved", saved.Results[0].Status)
		require.Equal(t, "saved", saved.Results[1].Status)

		var other state.Snapshot
		requestAs(t, "GET", "/?application=fixture-app", nil, actorID, "viewer", true, 200, &other)
		require.Equal(t, personal.Partition, other.Partition)
		require.Len(t, other.Entries, 2)
		requestAs(t, "GET", "/?application=other-app", nil, actorID, "viewer", true, 200, &other)
		require.Empty(t, other.Entries)
		requestAs(t, "GET", "/?application=fixture-app", nil, adminID, "admin", true, 200, &other)
		require.Empty(t, other.Entries)
		requestAs(t, "GET", target+"&application=fixture-app", nil, adminID, "admin", false, 400, nil)

		var separate state.Snapshot
		requestAs(t, "GET", "/", nil, actorID, "viewer", true, 200, &separate)
		change = state.Batch{ActorID: separate.ActorID, Partition: separate.Partition,
			Operations: []state.Operation{op("external.settings", `{"flag":true}`, 0)}}
		requestAs(t, "PATCH", "/", change, actorID, "viewer", true, 200, &saved)
		require.Equal(t, "saved", saved.Results[0].Status)
	})

	t.Run("SPP workspace", func(t *testing.T) {
		const key = "aodb.spp-tabs.workspace"
		const selected = `{"version":1,"tabs":[{"id":1,"kind":"arrival","filterStates":{"filterPanel":{"airport":"KZN"}}},{"id":2,"kind":"timeline"}],"activeId":2}`
		const empty = `{"version":1,"tabs":[],"activeId":null}`
		change := state.Batch{ActorID: own.ActorID, Partition: own.Partition,
			Operations: []state.Operation{op(key, selected, 0)}}
		var saved state.BatchResult
		requestAs(t, "PATCH", "/"+query, change, actorID, "viewer", false, 200, &saved)
		require.Equal(t, "saved", saved.Results[0].Status)
		entry := saved.Results[0].Entry
		require.JSONEq(t, selected, string(entry.Value))

		var managed state.Snapshot
		requestAs(t, "GET", target, nil, adminID, "editor", false, 200, &managed)
		require.Len(t, managed.Entries, 2)
		update := state.Batch{Partition: managed.Partition, ResetRevision: managed.ResetRevision,
			Operations: []state.Operation{op(key, `{"version":1,"tabs":[{"id":1,"kind":"unknown"}]}`, entry.Revision)}}
		requestAs(t, "PATCH", target, update, adminID, "editor", false, 400, nil)
		update.Operations = []state.Operation{op(key, empty, entry.Revision)}
		var edited state.BatchResult
		requestAs(t, "PATCH", target, update, adminID, "editor", false, 200, &edited)
		require.Equal(t, "saved", edited.Results[0].Status)
		var refreshed state.Snapshot
		requestAs(t, "GET", "/"+query, nil, actorID, "viewer", false, 200, &refreshed)
		for _, item := range refreshed.Entries {
			if item.Key == key {
				require.JSONEq(t, empty, string(item.Value))
				return
			}
		}
		t.Fatal("missing SPP workspace settings")
	})
}
