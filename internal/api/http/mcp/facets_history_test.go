package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/api/http/v1/auth_profile"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/computation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	"github.com/endge-lab/service-backend/internal/api/http/v1/update"
	"github.com/endge-lab/service-backend/internal/api/http/v1/vocab"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	validator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFormatVersionComesFromCreateRequest(t *testing.T) {
	cases := map[string]struct {
		request reflect.Type
		want    string
	}{
		entities.CollectionComponents:   {reflect.TypeFor[component.CreateRequest](), "modelVersion=1 (обязательно)"},
		entities.CollectionQueries:      {reflect.TypeFor[query.CreateRequest](), "sourceVersion=2 (обязательно, другое значение — ошибка)"},
		entities.CollectionVocabs:       {reflect.TypeFor[vocab.CreateRequest](), "sourceVersion=1 (обязательно вместе с source)"},
		entities.CollectionComputations: {reflect.TypeFor[computation.CreateRequest](), "sourceVersion=1 (обязательно); ещё обязательно: contractVersion=1"},
		entities.CollectionUpdates:      {reflect.TypeFor[update.CreateRequest](), "sourceVersion=1 (обязательно); ещё обязательно: storeIdentity"},
		entities.CollectionAuthProfiles: {reflect.TypeFor[auth_profile.CreateRequest](), "нет поля версии; ещё обязательно: adapterId"},
		entities.CollectionFolders:      {reflect.TypeFor[folder.CreateRequest](), "нет поля версии"},
	}
	for collection, tc := range cases {
		if got := formatVersionOf(collection, tc.request).describe(); got != tc.want {
			t.Fatalf("%s: got %q, want %q", collection, got, tc.want)
		}
	}
	contracts := BindContracts(validator.NewValidator())
	if len(contracts.FormatVersions) != len(entities.DocumentCollections) {
		t.Fatalf("every collection needs a format version entry: %d", len(contracts.FormatVersions))
	}
	for collection, version := range contracts.FormatVersions {
		if version.Field != "" && !slices.Contains(formatVersionFields, version.Field) {
			t.Fatalf("%s: unexpected field %s", collection, version.Field)
		}
	}
}

// Текущие версии формата совпадают с тем, что пишет Configurator при создании документа
// (CreateDocument_Modal) и что проверено компиляцией примеров agents/document-examples.md:
// Query — 2, остальные коллекции с source — 1, Component — modelVersion 1.
func TestFormatVersionCurrentValuesMatchConfigurator(t *testing.T) {
	contracts := BindContracts(validator.NewValidator())
	for collection, version := range contracts.FormatVersions {
		want := "1"
		if collection == entities.CollectionQueries {
			want = "2"
		}
		if version.Field != "" && version.Current != want {
			t.Fatalf("%s: current %s=%q, want %q", collection, version.Field, version.Current, want)
		}
		if version.Field == "" && documents.RequiresSourceVersion(collection) {
			t.Fatalf("%s: sourceVersion is required by the use case but missing in the DTO", collection)
		}
	}
	if got := contracts.FormatVersions[entities.CollectionComponents].Field; got != "modelVersion" {
		t.Fatalf("components use modelVersion, got %q", got)
	}
	if got := contracts.FormatVersions[entities.CollectionComputations].Also; !slices.Contains(got, "contractVersion=1") {
		t.Fatalf("computations require contractVersion=1: %v", got)
	}
}

func TestMCPCreateDescriptionNamesFormatVersionField(t *testing.T) {
	h := newHarness(t)
	for _, tc := range []struct {
		token     string
		authEnum  bool
		wantLines []string
	}{
		{"write", false, []string{"- components: modelVersion=1 (обязательно)", "- queries: sourceVersion=2 (обязательно, другое значение — ошибка)", "- types: sourceVersion=1 (обязательно)", "- computations: sourceVersion=1 (обязательно); ещё обязательно: contractVersion=1", "- Недоступно этому токену: auth-profiles", "auth: { mode: 'inherit' }"}},
		{"jwt-admin", true, []string{"- auth-profiles: нет поля версии; ещё обязательно: adapterId", "Персональному токену: auth-profiles"}},
	} {
		session := h.connect(t, tc.token)
		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		byName := map[string]*mcp.Tool{}
		for _, tool := range tools.Tools {
			byName[tool.Name] = tool
		}
		create := byName["endge_create_document"]
		for _, want := range append(tc.wantLines, "- folders: нет поля версии", "- facet-documents (с facetIdentity)") {
			if create == nil || !strings.Contains(create.Description, want) {
				t.Fatalf("%s: create description must contain %q: %v", tc.token, want, create)
			}
		}
		if strings.Contains(create.Description, "обычно") {
			t.Fatalf("%s: create description must name the current version, not a usual one: %s", tc.token, create.Description)
		}
		for _, name := range []string{"endge_create_document", "endge_update_document", "endge_restore_document", "endge_restore_revision"} {
			if got := schemaEnumHas(t, byName[name], "collection", entities.CollectionAuthProfiles); got != tc.authEnum {
				t.Fatalf("%s: %s offers auth-profiles=%v, want %v", tc.token, name, got, tc.authEnum)
			}
		}
		if !schemaEnumHas(t, byName["endge_get_document"], "collection", entities.CollectionAuthProfiles) {
			t.Fatalf("%s: reading auth-profiles stays available", tc.token)
		}
	}
	// Ошибка проверки тела называет текущее значение поля версии коллекции.
	session := h.connect(t, "write")
	invalid, isError := call(t, session, "endge_create_document", map[string]any{"collection": "queries", "document": map[string]any{"identity": "x", "displayName": "X", "source": "defineQuery({})", "modelVersion": 1}})
	if !isError || invalid["error"] != "validation_error" || !strings.Contains(invalid["hint"].(string), "sourceVersion=2 (обязательно, другое значение — ошибка)") {
		t.Fatalf("validation error must hint the format version field: %v", invalid)
	}
}

// schemaEnumHas сообщает, перечислено ли value в enum свойства property входной схемы инструмента.
func schemaEnumHas(t *testing.T, tool *mcp.Tool, property, value string) bool {
	t.Helper()
	if tool == nil {
		t.Fatalf("tool is missing")
	}
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	return slices.Contains(schema.Properties[property].Enum, value)
}

func TestMCPFacetsUseFacetUseCase(t *testing.T) {
	h := newHarness(t)
	reader := h.connect(t, "read")
	listed, isError := call(t, reader, "endge_list_documents", map[string]any{"collection": "facets"})
	items, _ := listed["items"].([]any)
	if isError || len(items) != 1 || items[0].(map[string]any)["icon"] != "Building" || items[0].(map[string]any)["documentCount"] != float64(1) {
		t.Fatalf("list facets: %v", listed)
	}
	missing, isError := call(t, reader, "endge_list_documents", map[string]any{"collection": "facet-documents"})
	if !isError || missing["error"] != "facet_identity_required" || missing["status"] != float64(400) || !strings.Contains(missing["hint"].(string), "facetIdentity") {
		t.Fatalf("facet documents need facetIdentity: %v", missing)
	}
	document, isError := call(t, reader, "endge_get_document", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "identity": "tenant-a"})
	if isError || document["facetIdentity"] != "tenant" || document["configuration"].(map[string]any)["region"] != "eu" {
		t.Fatalf("get facet document: %v", document)
	}
	revisions, isError := call(t, reader, "endge_document_revisions", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "identity": "tenant-a", "limit": 1})
	if isError || revisions["total"] != float64(2) || len(revisions["items"].([]any)) != 1 {
		t.Fatalf("facet document revisions must be paged in MCP: %v", revisions)
	}

	writer := h.connect(t, "write")
	invalid, isError := call(t, writer, "endge_create_document", map[string]any{"collection": "facets", "document": map[string]any{"identity": "region", "displayName": "Регион", "icon": "MapPin", "color": "blue"}})
	if !isError || invalid["error"] != "validation_error" {
		t.Fatalf("facet DTO must validate color length: %v", invalid)
	}
	created, isError := call(t, writer, "endge_create_document", map[string]any{"collection": "facets", "document": map[string]any{"identity": "region", "displayName": "Регион", "icon": "MapPin", "color": "#2563eb"}})
	if isError || created["revision"] != float64(1) {
		t.Fatalf("create facet: %v", created)
	}
	entry, isError := call(t, writer, "endge_create_document", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "document": map[string]any{"identity": "tenant-c", "displayName": "Тенант C", "configuration": map[string]any{"mode": "inherit", "patch": map[string]any{}}}})
	if isError || entry["identity"] != "tenant-c" || entry["facetIdentity"] != "tenant" {
		t.Fatalf("create facet document: %v", entry)
	}
	conflict, isError := call(t, writer, "endge_update_document", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "identity": "tenant-a", "revision": 1, "changes": map[string]any{"displayName": "A"}})
	if !isError || conflict["error"] != "revision_conflict" {
		t.Fatalf("stale facet document revision: %v", conflict)
	}
	restored, isError := call(t, writer, "endge_restore_revision", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "identity": "tenant-a", "revisionId": "fr-1", "revision": 2})
	if isError || restored["revision"] != float64(3) {
		t.Fatalf("restore facet document revision: %v", restored)
	}
	if _, err := writer.CallTool(context.Background(), &mcp.CallToolParams{Name: "endge_delete_document", Arguments: map[string]any{"collection": "facets", "identity": "tenant", "revision": 1}}); err == nil {
		t.Fatalf("write token without delete must not delete facets")
	}
	full := h.connect(t, "full")
	deleted, isError := call(t, full, "endge_delete_document", map[string]any{"collection": "facet-documents", "facetIdentity": "tenant", "identity": "tenant-a", "revision": 2})
	if isError || deleted["deleted"] != true {
		t.Fatalf("delete facet document: %v", deleted)
	}
	want := []string{"List", "GetDocument tenant tenant-a", "ListDocumentRevisions tenant tenant-a", "Create region #2563eb", "CreateDocument tenant tenant-c", "RestoreDocumentRevision tenant tenant-a fr-1", "DeleteDocument tenant tenant-a"}
	if !slices.Equal(h.facets.calls, want) {
		t.Fatalf("facet use case calls = %v, want %v", h.facets.calls, want)
	}
}

func TestMCPRevisionReadAndRestore(t *testing.T) {
	h := newHarness(t)
	reader := h.connect(t, "read")
	revision, isError := call(t, reader, "endge_get_revision", map[string]any{"collection": "queries", "identity": "flights", "revisionId": "r-1"})
	if isError || revision["revisionNumber"] != float64(1) || revision["snapshot"].(map[string]any)["displayName"] != "Рейсы" {
		t.Fatalf("get revision must include the snapshot: %v", revision)
	}
	notFound, isError := call(t, reader, "endge_get_revision", map[string]any{"collection": "queries", "identity": "flights", "revisionId": "missing"})
	if !isError || notFound["status"] != float64(404) {
		t.Fatalf("missing revision: %v", notFound)
	}
	writer := h.connect(t, "write")
	stale, isError := call(t, writer, "endge_restore_revision", map[string]any{"collection": "queries", "identity": "flights", "revisionId": "r-1", "revision": 5})
	if !isError || stale["error"] != "revision_conflict" {
		t.Fatalf("restore with stale revision: %v", stale)
	}
	restored, isError := call(t, writer, "endge_restore_revision", map[string]any{"collection": "queries", "identity": "flights", "revisionId": "r-1", "revision": 1})
	if isError || restored["revision"] != float64(2) || !slices.Equal(h.revisions.restored, []string{"queries/flights@r-1"}) {
		t.Fatalf("restore revision: %v %v", restored, h.revisions.restored)
	}
	// Прямой вызов обработчика с токеном на чтение отклоняет use case.
	tools, _ := newTestTools()
	ctx := entities.WithAccessToken(context.Background(), entities.AccessTokenPrincipal{TokenID: "t", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeRead})
	result, _, _ := tools.restoreRevision(ctx, nil, restoreRevisionInput{Collection: "queries", Identity: "flights", RevisionID: "r-1", Revision: 1})
	if body := decodeResult(t, result); !result.IsError || body["error"] != "access_token_scope_insufficient" {
		t.Fatalf("read token restore: %v", body)
	}
}

func TestMCPCommitsAndPlanDocuments(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "read")
	listed, isError := call(t, session, "endge_list_commits", map[string]any{"limit": 1})
	items, _ := listed["items"].([]any)
	if isError || listed["total"] != float64(2) || len(items) != 1 || items[0].(map[string]any)["id"] != "c-2" || items[0].(map[string]any)["changes"] != nil {
		t.Fatalf("list commits must be a page without changes: %v", listed)
	}
	second, _ := call(t, session, "endge_list_commits", map[string]any{"limit": 1, "offset": 1})
	if second["items"].([]any)[0].(map[string]any)["id"] != "c-1" {
		t.Fatalf("second page: %v", second)
	}
	diff, isError := call(t, session, "endge_get_commit", map[string]any{"id": "c-2"})
	changes, _ := diff["changes"].([]any)
	if isError || len(changes) != 1 || changes[0].(map[string]any)["afterRevisionId"] != "r-2" || changes[0].(map[string]any)["documentIdentity"] != "flights" {
		t.Fatalf("commit diff: %v", diff)
	}
	missing, isError := call(t, session, "endge_get_commit", map[string]any{"id": "nope"})
	if !isError || missing["error"] != "commit_not_found" {
		t.Fatalf("missing commit: %v", missing)
	}
	other, isError := call(t, session, "endge_list_commits", map[string]any{"workspace": "kzn"})
	if !isError || other["error"] != "access_token_workspace_forbidden" {
		t.Fatalf("commits are limited to the token workspace: %v", other)
	}
	plan, isError := call(t, session, "endge_commit_plan", nil)
	documents, _ := plan["documents"].([]any)
	if isError || len(documents) != 1 {
		t.Fatalf("plan must list documents: %v", plan)
	}
	entry := documents[0].(map[string]any)
	if entry["collection"] != "queries" || entry["identity"] != "flights" || entry["displayName"] != "Рейсы" || entry["revisionCount"] != float64(2) || entry["accessTokens"].([]any)[0].(map[string]any)["name"] != "Codex" {
		t.Fatalf("plan document: %v", entry)
	}
}
