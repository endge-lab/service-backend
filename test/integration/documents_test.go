//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres"
	"github.com/endge-lab/service-backend/internal/usecase/commits"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/release_artifacts"
	"github.com/endge-lab/service-backend/internal/usecase/releases"
	"github.com/endge-lab/service-backend/internal/usecase/revisions"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
	"github.com/endge-lab/service-backend/test/support"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

type repositoryFixture struct {
	database   *support.TestDatabase
	ctx        context.Context
	store      *postgres.EndgeRepository
	tx         *postgres.TxManager
	lifecycle  *documents.Lifecycle
	workspaces *workspaces.UseCase
	revisions  *revisions.UseCase
	commits    *commits.UseCase
	releases   *releases.UseCase
	resources  map[string]ports.DocumentResourceRepository
}

// TestEveryDocumentRepositoryLifecycle проверяет общий CRUD и историю всех 22 таблиц документов.
func TestEveryDocumentRepositoryLifecycle(t *testing.T) {
	fixture := newRepositoryFixture(t)
	cases := documentCases()
	created := make(map[string]*entities.Document, len(cases))

	for _, testCase := range cases {
		t.Run("создание "+testCase.collection, func(t *testing.T) {
			input := createInput(t, testCase.payload)
			value, err := fixture.lifecycle.Create(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], input)
			if err != nil {
				t.Fatalf("создать %s: %v", testCase.collection, err)
			}
			if value.Revision != 1 || value.CreatedBy.ID == "" || value.UpdatedBy.ID != value.CreatedBy.ID {
				t.Fatalf("неверные audit/revision у %s: %#v", testCase.collection, value)
			}
			created[testCase.collection] = value
		})
	}

	for _, testCase := range cases {
		t.Run("изменение и восстановление "+testCase.collection, func(t *testing.T) {
			initial := created[testCase.collection]
			noOp := patchInput(t, map[string]any{"displayName": initial.DisplayName})
			unchanged, err := fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], initial.Identity, noOp, initial.Revision)
			if err != nil || unchanged.Revision != initial.Revision {
				t.Fatalf("no-op %s создал revision: value=%#v err=%v", testCase.collection, unchanged, err)
			}
			changed, err := fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], initial.Identity, patchInput(t, map[string]any{"displayName": initial.DisplayName + " updated"}), initial.Revision)
			if err != nil || changed.Revision != 2 {
				t.Fatalf("patch %s: revision=%v err=%v", testCase.collection, changed, err)
			}
			if _, err = fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], initial.Identity, patchInput(t, map[string]any{"displayName": "stale"}), initial.Revision); err == nil {
				t.Fatalf("stale patch %s был принят", testCase.collection)
			}
			deleted, err := fixture.lifecycle.Delete(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], initial.Identity, changed.Revision)
			if err != nil || deleted.DeletedAt == nil || deleted.Revision != 3 {
				t.Fatalf("soft-delete %s: value=%#v err=%v", testCase.collection, deleted, err)
			}
			visible, err := fixture.resources[testCase.collection].List(fixture.ctx, deleted.WorkspaceID, ports.DocumentFilter{Limit: 500})
			if err != nil {
				t.Fatalf("list %s: %v", testCase.collection, err)
			}
			for _, item := range visible {
				if item.Identity == initial.Identity {
					t.Fatalf("удалённый %s присутствует в обычном list", testCase.collection)
				}
			}
			restored, err := fixture.lifecycle.Restore(fixture.ctx, documents.Definition{Collection: testCase.collection}, fixture.resources[testCase.collection], initial.Identity, deleted.Revision)
			if err != nil || restored.DeletedAt != nil || restored.Revision != 4 {
				t.Fatalf("restore %s: value=%#v err=%v", testCase.collection, restored, err)
			}
			historyItems, err := fixture.revisions.List(fixture.ctx, testCase.collection, initial.Identity)
			if err != nil || len(historyItems) != 4 {
				t.Fatalf("history %s: revisions=%d err=%v", testCase.collection, len(historyItems), err)
			}
		})
	}
}

// TestTransactionRollbackLeavesNoPartialDocument проверяет настоящий rollback PostgreSQL transaction.
func TestTransactionRollbackLeavesNoPartialDocument(t *testing.T) {
	fixture := newRepositoryFixture(t)
	scope, _ := entities.WorkspaceAccessFromContext(fixture.ctx)
	actor, _ := entities.CurrentActorFromContext(fixture.ctx)
	document := entities.Document{
		ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, Type: "queries", Identity: "rolled-back-query",
		DisplayName: "Rollback", ManagedBy: "user", Meta: json.RawMessage(`{}`), Data: json.RawMessage(`{"source":"query {}","sourceVersion":2}`),
		Active: true, Revision: 1, CreatedBy: entities.Actor{ID: actor.User.ID}, UpdatedBy: entities.Actor{ID: actor.User.ID},
	}
	expected := errors.New("искусственная ошибка после insert")
	err := fixture.tx.WithinTransaction(fixture.ctx, func(txctx context.Context) error {
		folderID, resolveErr := fixture.store.ResolveFolder(txctx, scope.Workspace.ID, "root-queries", "queries")
		if resolveErr != nil {
			return resolveErr
		}
		if _, insertErr := fixture.store.InsertDocument(txctx, document, folderID); insertErr != nil {
			return insertErr
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("transaction вернула %v, ожидалась искусственная ошибка", err)
	}
	var count int
	if err = fixture.database.Pool.QueryRow(context.Background(), `SELECT count(*) FROM queries WHERE identity='rolled-back-query'`).Scan(&count); err != nil {
		t.Fatalf("проверить rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("после rollback осталось документов: %d", count)
	}
}

// TestCommitSquashReleaseAndRestore проверяет критическую цепочку истории workspace.
func TestCommitSquashReleaseAndRestore(t *testing.T) {
	fixture := newRepositoryFixture(t)
	queryRepository := fixture.resources["queries"]
	created, err := fixture.lifecycle.Create(fixture.ctx, documents.Definition{Collection: "queries"}, queryRepository, createInput(t, map[string]any{
		"identity": "history-query", "displayName": "History Query", "source": "query {}", "sourceVersion": 2,
	}))
	if err != nil {
		t.Fatalf("создать query: %v", err)
	}
	updated, err := fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: "queries"}, queryRepository, created.Identity, patchInput(t, map[string]any{"source": "query { updated }"}), created.Revision)
	if err != nil {
		t.Fatalf("изменить query: %v", err)
	}
	scope, err := fixture.workspaces.Authorize(fixture.ctx, "default")
	if err != nil {
		t.Fatalf("обновить workspace scope: %v", err)
	}
	fixture.ctx = entities.WithWorkspaceAccess(fixture.ctx, scope)
	commit, err := fixture.commits.Create(fixture.ctx, "Preserved history", "preserve", scope.Workspace.HeadSequence)
	if err != nil {
		t.Fatalf("создать preserve commit: %v", err)
	}
	release, err := fixture.releases.Create(fixture.ctx, releases.CreateInput{Identity: "release-one", DisplayName: "Release One", SourceCommitID: commit.ID})
	if err != nil || release.Checksum == "" {
		t.Fatalf("создать release: value=%#v err=%v", release, err)
	}
	artifact, err := fixture.store.GetReleaseArtifact(fixture.ctx, release.WorkspaceID, release.ID)
	if err != nil || len(artifact.Data) == 0 {
		t.Fatalf("прочитать release artifact: value=%#v err=%v", artifact, err)
	}

	third, err := fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: "queries"}, queryRepository, created.Identity, patchInput(t, map[string]any{"source": "query { third }"}), updated.Revision)
	if err != nil {
		t.Fatalf("третья revision: %v", err)
	}
	_, err = fixture.lifecycle.Patch(fixture.ctx, documents.Definition{Collection: "queries"}, queryRepository, created.Identity, patchInput(t, map[string]any{"source": "query { fourth }"}), third.Revision)
	if err != nil {
		t.Fatalf("четвёртая revision: %v", err)
	}
	scope, _ = fixture.workspaces.Authorize(fixture.ctx, "default")
	fixture.ctx = entities.WithWorkspaceAccess(fixture.ctx, scope)
	squashedCommit, err := fixture.commits.Create(fixture.ctx, "Squashed history", "squash", scope.Workspace.HeadSequence)
	if err != nil {
		t.Fatalf("создать squash commit: %v", err)
	}
	secondRelease, err := fixture.releases.Create(fixture.ctx, releases.CreateInput{Identity: "release-two", DisplayName: "Release Two", SourceCommitID: squashedCommit.ID})
	if err != nil {
		t.Fatalf("создать второй release: %v", err)
	}
	last, err := fixture.releases.Get(fixture.ctx, "last")
	if err != nil || last.ID != secondRelease.ID {
		t.Fatalf("last не указывает на новый release: value=%#v err=%v", last, err)
	}
	revisionItems, err := fixture.revisions.List(fixture.ctx, "queries", created.Identity)
	if err != nil {
		t.Fatalf("получить revisions после squash: %v", err)
	}
	if len(revisionItems) != 3 || revisionItems[0].Operation != "squash" || len(revisionItems[0].Contributors) != 1 {
		t.Fatalf("неверный squash result: %#v", revisionItems)
	}

	scope, _ = fixture.workspaces.Authorize(fixture.ctx, "default")
	fixture.ctx = entities.WithWorkspaceAccess(fixture.ctx, scope)
	restoreCommit, err := fixture.releases.Restore(fixture.ctx, release.Identity, scope.Workspace.HeadSequence)
	if err != nil || restoreCommit.Operation != "release_restore" {
		t.Fatalf("восстановить release: commit=%#v err=%v", restoreCommit, err)
	}
	restored, err := queryRepository.Get(fixture.ctx, scope.Workspace.ID, created.Identity, false)
	if err != nil || string(restored.Data) != string(updated.Data) {
		t.Fatalf("release restore не вернул snapshot: value=%#v err=%v", restored, err)
	}
}

func newRepositoryFixture(t *testing.T) *repositoryFixture {
	t.Helper()
	database := postgresSuite.NewDatabase(t)
	userID := uuid.NewString()
	if _, err := database.Pool.Exec(context.Background(), `INSERT INTO service_users(id,provider_id,subject,issuer,username,display_name) VALUES($1,'integration',$2,'urn:endge:test','tester','Integration Tester')`, userID, "subject-"+userID); err != nil {
		t.Fatalf("создать тестового пользователя: %v", err)
	}
	store := postgres.NewEndgeRepository(database.Pool, 1)
	tx := postgres.NewTxManager(database.Pool, observability.NewCore(nil, zap.NewNop()), nil)
	recorder := history.NewRecorder(store)
	cfg := support.DevConfig()
	artifacts, err := release_artifacts.NewReader(store, cfg.ReleaseArtifactCache, noop.NewMeterProvider().Meter("integration"))
	if err != nil {
		t.Fatalf("создать reader artifact: %v", err)
	}
	coordinator := workspace_state.NewCoordinator(store, tx, artifacts, 1)
	lifecycle := documents.NewLifecycle(store, tx, recorder)
	workspaceUseCase := workspaces.NewUseCase(store, store, store, tx, recorder)
	actor := entities.CurrentActor{User: &entities.User{ID: userID, ProviderID: "integration", Subject: "subject-" + userID, Issuer: "urn:endge:test", Username: "tester", DisplayName: "Integration Tester", Active: true}, PlatformAdmin: true}
	ctx := entities.WithCurrentActor(context.Background(), actor)
	scope, err := workspaceUseCase.Authorize(ctx, "default")
	if err != nil {
		t.Fatalf("авторизовать default workspace: %v", err)
	}
	ctx = entities.WithWorkspaceAccess(ctx, scope)
	return &repositoryFixture{
		database: database, ctx: ctx, store: store, tx: tx, lifecycle: lifecycle, workspaces: workspaceUseCase,
		revisions: revisions.NewUseCase(store, coordinator), commits: commits.NewUseCase(store, store, tx, coordinator),
		releases: releases.NewUseCase(store, store, store, coordinator, artifacts), resources: documentRepositories(store),
	}
}

type documentCase struct {
	collection string
	payload    map[string]any
}

func documentCases() []documentCase {
	return []documentCase{
		{collection: "environments", payload: baseDocument("environment-main")},
		{collection: "stores", payload: with(baseDocument("store-main"), "source", "store {}", "sourceVersion", 1)},
		{collection: "auth-profiles", payload: with(baseDocument("auth-main"), "adapterId", "oidc", "config", map[string]any{"issuer": "https://issuer.example", "clientId": "endge-test", "scopes": []any{"openid"}}, "credentials", map[string]any{}, "session", map[string]any{"storage": "memory", "persistRefreshToken": false})},
		{collection: "projects", payload: with(baseDocument("project-main"), "allowedEnvironments", []any{"environment-main"})},
		{collection: "tenants", payload: with(baseDocument("tenant-main"), "code", "TENANT")},
		{collection: "folders", payload: with(baseDocument("folder-main"), "entityType", "queries")},
		{collection: "types", payload: with(baseDocument("type-main"), "source", "type {}", "sourceVersion", 1)},
		{collection: "queries", payload: with(baseDocument("query-main"), "source", "query {}", "sourceVersion", 2)},
		{collection: "data-views", payload: with(baseDocument("data-view-main"), "source", "view {}", "sourceVersion", 1)},
		{collection: "compositions", payload: with(baseDocument("composition-main"), "kind", "screen", "kindIdentity", "main", "source", "composition {}", "sourceVersion", 1)},
		{collection: "streams", payload: with(baseDocument("stream-main"), "source", "stream {}", "sourceVersion", 1)},
		{collection: "updates", payload: with(baseDocument("update-main"), "storeIdentity", "store-main", "source", "update {}", "sourceVersion", 1)},
		{collection: "mocks", payload: with(baseDocument("mock-main"), "contentSource", "inline", "contentType", "application/json", "source", "{}")},
		{collection: "components", payload: with(baseDocument("component-main"), "source", "<template />", "tag", "endge-main", "modelVersion", 1, "supportedTargets", []any{"vue"})},
		{collection: "actions", payload: with(baseDocument("action-main"), "definition", map[string]any{}, "input", map[string]any{}, "output", map[string]any{}, "target", "runtime")},
		{collection: "filters", payload: with(baseDocument("filter-main"), "fields", []any{}, "source", "filter {}", "sourceVersion", 1)},
		{collection: "converters", payload: with(baseDocument("converter-main"), "definition", map[string]any{})},
		{collection: "computations", payload: with(baseDocument("computation-main"), "source", "compute {}", "sourceVersion", 1, "contractVersion", 1)},
		{collection: "vocabs", payload: with(baseDocument("vocab-main"), "source", "defineVocab({ outputs: { items: output().from(response()) } })", "sourceVersion", 1, "mode", "external_payload", "authMode", "profile", "authProfileIdentity", "auth-main")},
		{collection: "i18n-bundles", payload: with(baseDocument("i18n-main"), "locales", map[string]any{"ru": map[string]any{"title": "Тест"}})},
		{collection: "navigations", payload: with(baseDocument("navigation-main"), "tree", []any{})},
		{collection: "styles", payload: with(baseDocument("style-main"), "source", "body {}", "sourceVersion", 1)},
		{collection: "configurations", payload: with(baseDocument("configuration-main"), "source", "defineConfig({ enabled: value(Boolean, true) })", "sourceVersion", 1)},
	}
}

func documentRepositories(store *postgres.EndgeRepository) map[string]ports.DocumentResourceRepository {
	return map[string]ports.DocumentResourceRepository{
		"projects": postgres.NewProjectRepository(store), "tenants": postgres.NewTenantRepository(store),
		"environments": postgres.NewEnvironmentRepository(store), "folders": postgres.NewFolderRepository(store),
		"types": postgres.NewTypeRepository(store), "queries": postgres.NewQueryRepository(store),
		"data-views": postgres.NewDataViewRepository(store), "compositions": postgres.NewCompositionRepository(store),
		"stores": postgres.NewStoreRepository(store), "streams": postgres.NewStreamRepository(store),
		"updates": postgres.NewUpdateRepository(store), "mocks": postgres.NewMockRepository(store),
		"components": postgres.NewComponentRepository(store), "actions": postgres.NewActionRepository(store),
		"filters": postgres.NewFilterRepository(store), "converters": postgres.NewConverterRepository(store),
		"computations": postgres.NewComputationRepository(store), "vocabs": postgres.NewVocabRepository(store),
		"i18n-bundles": postgres.NewI18nBundleRepository(store), "auth-profiles": postgres.NewAuthProfileRepository(store),
		"navigations": postgres.NewNavigationRepository(store), "styles": postgres.NewStyleRepository(store), "configurations": postgres.NewConfigurationRepository(store),
	}
}

func baseDocument(identity string) map[string]any {
	return map[string]any{"identity": identity, "displayName": identity}
}

func with(value map[string]any, pairs ...any) map[string]any {
	for index := 0; index < len(pairs); index += 2 {
		value[fmt.Sprint(pairs[index])] = pairs[index+1]
	}
	return value
}

func createInput(t *testing.T, value map[string]any) documents.CreateInput {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("сериализовать create input: %v", err)
	}
	input, err := documents.NewCreateInputJSON(raw)
	if err != nil {
		t.Fatalf("создать create input: %v", err)
	}
	return input
}

func patchInput(t *testing.T, value map[string]any) documents.PatchInput {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("сериализовать patch input: %v", err)
	}
	input, err := documents.NewPatchInputJSON(raw)
	if err != nil {
		t.Fatalf("создать patch input: %v", err)
	}
	return input
}
