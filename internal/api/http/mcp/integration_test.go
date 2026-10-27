package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	validator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// testPrincipals — токены, которые понимает поддельная аутентификация; "jwt", "jwt-admin"
// и "jwt-platform" — сессии пользователя (Editor/Viewer, Admin в afl, Platform Admin).
var testPrincipals = map[string]entities.AccessTokenPrincipal{
	"read":  {TokenID: "t-read", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeRead, ExpiresAt: time.Unix(4102444800, 0)},
	"write": {TokenID: "t-write", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeWrite, ExpiresAt: time.Unix(4102444800, 0)},
	"full":  {TokenID: "t-full", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeWrite, CanCommit: true, CanDelete: true, ExpiresAt: time.Unix(4102444800, 0)},
	// Флаг commit есть, но владелец — Editor: endge_commit скрыт.
	"full-editor": {TokenID: "t-full-editor", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeWrite, CanCommit: true, CanDelete: true, ExpiresAt: time.Unix(4102444800, 0)},
}

// fakeAuth заменяет AuthMiddleware и CurrentUserMiddleware: кладёт в context те же значения.
func fakeAuth(c *fiber.Ctx) error {
	value := strings.TrimPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	actor := entities.CurrentActor{User: testUser}
	switch value {
	case "jwt-admin":
		actor.User = testAdminUser
	case "jwt-platform":
		actor.PlatformAdmin = true
	}
	ctx := entities.WithCurrentActor(c.UserContext(), actor)
	if principal, ok := testPrincipals[value]; ok {
		ctx = entities.WithAccessToken(ctx, principal)
	} else if !strings.HasPrefix(value, "jwt") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "authentication required"})
	}
	c.SetUserContext(ctx)
	return c.Next()
}

type headerTransport struct{ token string }

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(r)
}

type harness struct {
	server *httptest.Server
	docs   *httptest.Server
	fakes
}

// newHarness поднимает Fiber с маршрутом /endge/mcp (HTTP_BASE_PATH) за httptest-сервером.
func newHarness(t *testing.T) *harness {
	t.Helper()
	useCases, fakes := newFakeUseCases()
	docs := newDocsServer(t)
	handler := NewHandler(Config{Service: "endge-backend", Version: "0.16.0", Env: "test", AgentDocsURL: docs.URL + "/endge-docs"}, useCases, BindContracts(validator.NewValidator()), zap.NewNop())
	app := fiber.New()
	RegisterRoutes(app.Group("/endge"), handler, fakeAuth, func(c *fiber.Ctx) error { return c.Next() })
	server := httptest.NewServer(adaptor.FiberApp(app))
	t.Cleanup(server.Close)
	return &harness{server: server, docs: docs, fakes: fakes}
}

func (h *harness) connect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "endge-test", Version: "1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: h.server.URL + "/endge/mcp", HTTPClient: &http.Client{Transport: headerTransport{token: token}}, DisableStandaloneSSE: true}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect as %s: %v", token, err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := []string{}
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

// call вызывает инструмент и разбирает JSON-текст результата.
func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	value := map[string]any{}
	if err = json.Unmarshal([]byte(text), &value); err != nil {
		// Ошибки проверки входа по схеме SDK возвращает обычным текстом.
		return map[string]any{"text": text}, result.IsError
	}
	return value, result.IsError
}

func TestMCPRejectsMissingTokenWithChallenge(t *testing.T) {
	h := newHarness(t)
	request, _ := http.NewRequest(http.MethodPost, h.server.URL+"/endge/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(response.Header.Get("WWW-Authenticate"), `Bearer realm="endge"`) {
		t.Fatalf("expected 401 with Bearer challenge, got %d %q", response.StatusCode, response.Header.Get("WWW-Authenticate"))
	}
	if strings.Contains(response.Header.Get("WWW-Authenticate"), "invalid_token") {
		t.Fatalf("challenge without credentials must not report invalid_token")
	}
}

func TestMCPIsStatelessAndRejectsGet(t *testing.T) {
	h := newHarness(t)
	request, _ := http.NewRequest(http.MethodGet, h.server.URL+"/endge/mcp", nil)
	request.Header.Set("Authorization", "Bearer read")
	request.Header.Set("Accept", "text/event-stream")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("stateless MCP must answer GET with 405, got %d", response.StatusCode)
	}
	session := h.connect(t, "read")
	if session.ID() != "" {
		t.Fatalf("stateless server must not issue Mcp-Session-Id, got %q", session.ID())
	}
	if !strings.Contains(session.InitializeResult().Instructions, "ресурс llms-agent ("+h.docs.URL+"/endge-docs/llms-agent.txt)") {
		t.Fatalf("instructions must point to the agent core first: %s", session.InitializeResult().Instructions)
	}
}

func TestMCPReadTokenSeesOnlyReadTools(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "read")
	want := []string{"endge_commit_plan", "endge_document_revisions", "endge_get_commit", "endge_get_document", "endge_get_revision", "endge_get_workspace", "endge_list_commits", "endge_list_documents", "endge_list_workspaces", "endge_status"}
	if names := toolNames(t, session); !slices.Equal(names, want) {
		t.Fatalf("read token tools = %v, want %v", names, want)
	}
	status, _ := call(t, session, "endge_status", nil)
	if status["authentication"] != "access_token" || status["defaultWorkspace"] != "afl" || status["version"] != "0.16.0" {
		t.Fatalf("unexpected status: %v", status)
	}
	listed, isError := call(t, session, "endge_list_documents", map[string]any{"collection": "queries"})
	items, _ := listed["items"].([]any)
	if isError || len(items) != 1 || items[0].(map[string]any)["source"] != nil || listed["limit"] != float64(100) {
		t.Fatalf("list must return a summary page with default limit: %v", listed)
	}
	document, _ := call(t, session, "endge_get_document", map[string]any{"collection": "queries", "identity": "flights"})
	if document["source"] != "defineQuery({})" || document["revision"] != float64(1) {
		t.Fatalf("get must return the full document: %v", document)
	}
	snapshot, _ := call(t, session, "endge_get_workspace", nil)
	if workspace := snapshot["workspace"].(map[string]any); workspace["secret"] != nil || workspace["identity"] != "afl" {
		t.Fatalf("workspace summary must keep only public fields: %v", workspace)
	}
	revisions, _ := call(t, session, "endge_document_revisions", map[string]any{"collection": "queries", "identity": "flights"})
	first := revisions["items"].([]any)[0].(map[string]any)
	if first["accessTokenName"] != "Codex" || first["snapshot"] != nil || h.revisions.last.Limit != 20 || h.revisions.last.IncludeSnapshot {
		t.Fatalf("revisions must be a page without snapshots: %v %+v", first, h.revisions.last)
	}
	other, isError := call(t, session, "endge_list_documents", map[string]any{"collection": "queries", "workspace": "kzn"})
	if !isError || other["error"] != "access_token_workspace_forbidden" || other["status"] != float64(403) || other["hint"] == "" {
		t.Fatalf("token must be limited to its workspace: %v", other)
	}
	// Скрытый инструмент нельзя вызвать по имени.
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "endge_create_document", Arguments: map[string]any{"collection": "queries", "document": map[string]any{}}}); err == nil {
		t.Fatalf("read token must not call write tools")
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "endge_restore_revision", Arguments: map[string]any{"collection": "queries", "identity": "flights", "revisionId": "r-1", "revision": 1}}); err == nil {
		t.Fatalf("read token must not restore revisions")
	}
}

func TestMCPWriteTokenScenario(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "write")
	names := toolNames(t, session)
	if !slices.Contains(names, "endge_update_document") || slices.Contains(names, "endge_delete_document") || slices.Contains(names, "endge_commit") {
		t.Fatalf("write token without flags tools = %v", names)
	}
	created, isError := call(t, session, "endge_create_document", map[string]any{"collection": "queries", "document": map[string]any{"identity": "created", "displayName": "Создан", "source": "defineQuery({})", "sourceVersion": 2}})
	if isError || created["revision"] != float64(1) {
		t.Fatalf("create: %v", created)
	}
	updated, isError := call(t, session, "endge_update_document", map[string]any{"collection": "queries", "identity": "flights", "revision": 1, "changes": map[string]any{"displayName": "Рейсы 2"}})
	if isError || updated["revision"] != float64(2) {
		t.Fatalf("update: %v", updated)
	}
	conflict, isError := call(t, session, "endge_update_document", map[string]any{"collection": "queries", "identity": "flights", "revision": 1, "changes": map[string]any{"displayName": "X"}})
	if !isError || conflict["error"] != "revision_conflict" || conflict["status"] != float64(409) || !strings.Contains(conflict["hint"].(string), "перечитайте") {
		t.Fatalf("stale revision must be a revision_conflict: %v", conflict)
	}
	if details := conflict["details"].(map[string]any); details["currentRevision"] != float64(2) {
		t.Fatalf("conflict must keep backend details: %v", conflict)
	}
	plan, isError := call(t, session, "endge_commit_plan", nil)
	if isError || plan["headSequence"] != float64(7) {
		t.Fatalf("plan: %v", plan)
	}
	// Третья запись — попытка с устаревшей revision: use case получил тот же токен и отказал.
	if !slices.Equal(h.documents.marks, []string{"t-write", "t-write", "t-write"}) {
		t.Fatalf("writes must carry the access token in context: %v", h.documents.marks)
	}
}

func TestMCPHidesCommitWhenOwnerIsNotAdmin(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "full-editor")
	names := toolNames(t, session)
	if slices.Contains(names, "endge_commit") || !slices.Contains(names, "endge_delete_document") || !slices.Contains(names, "endge_commit_plan") {
		t.Fatalf("commit must be hidden for an Editor owner: %v", names)
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "endge_commit", Arguments: map[string]any{"message": "агент", "expectedHeadSequence": 7}}); err == nil {
		t.Fatalf("hidden endge_commit must not be callable")
	}
	if len(h.commits.created) != 0 {
		t.Fatalf("commit must not reach the use case: %v", h.commits.created)
	}
	if names := toolNames(t, h.connect(t, "full")); !slices.Contains(names, "endge_commit") {
		t.Fatalf("Admin owner must see endge_commit: %v", names)
	}
}

func TestMCPFullTokenDeletesAndCommits(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "full")
	deleted, isError := call(t, session, "endge_delete_document", map[string]any{"collection": "queries", "identity": "flights", "revision": 1})
	if isError || deleted["deleted"] != true {
		t.Fatalf("delete: %v", deleted)
	}
	stale, isError := call(t, session, "endge_commit", map[string]any{"message": "агент", "expectedHeadSequence": 3})
	if !isError || stale["error"] != "head_sequence_conflict" {
		t.Fatalf("stale head must conflict: %v", stale)
	}
	committed, isError := call(t, session, "endge_commit", map[string]any{"message": "агент", "expectedHeadSequence": 7})
	if isError || committed["id"] != "c-1" || !slices.Equal(h.commits.created, []string{"агент/preserve"}) {
		t.Fatalf("commit: %v %v", committed, h.commits.created)
	}
	// Вход endge_commit проверяет DTO REST commit.CreateRequest, а не схема SDK.
	for name, args := range map[string]map[string]any{
		"empty message":  {"message": "", "expectedHeadSequence": 7},
		"long message":   {"message": strings.Repeat("я", 1001), "expectedHeadSequence": 7},
		"negative head":  {"message": "агент", "expectedHeadSequence": -1},
		"unknown policy": {"message": "агент", "expectedHeadSequence": 7, "revisionPolicy": "rebase"},
	} {
		invalid, isError := call(t, session, "endge_commit", args)
		if !isError || invalid["error"] != "validation_error" || invalid["status"] != float64(400) || invalid["details"] == nil || invalid["hint"] == "" {
			t.Fatalf("%s must be a REST validation_error: %v", name, invalid)
		}
	}
	if len(h.commits.created) != 1 {
		t.Fatalf("invalid commits must not reach the use case: %v", h.commits.created)
	}
}

// TestMCPValidatesDocumentsWithRESTContracts проверяет, что create/update документа
// отклоняются теми же правилами DTO, что REST (обязательные поля, длины, enum,
// неизвестные поля), до вызова use case.
func TestMCPValidatesDocumentsWithRESTContracts(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "write")
	creates := map[string]map[string]any{
		"missing displayName": {"identity": "x", "source": "defineQuery({})", "sourceVersion": 2},
		"long identity":       {"identity": strings.Repeat("a", 161), "displayName": "X", "source": "defineQuery({})", "sourceVersion": 2},
		"wrong sourceVersion": {"identity": "x", "displayName": "X", "source": "defineQuery({})", "sourceVersion": 1},
		"unknown field":       {"identity": "x", "displayName": "X", "source": "defineQuery({})", "sourceVersion": 2, "legacy": true},
	}
	for name, document := range creates {
		invalid, isError := call(t, session, "endge_create_document", map[string]any{"collection": "queries", "document": document})
		if !isError || invalid["error"] != "validation_error" || invalid["status"] != float64(400) {
			t.Fatalf("create %s must be a REST validation_error: %v", name, invalid)
		}
	}
	missing, _ := call(t, session, "endge_create_document", map[string]any{"collection": "queries", "document": creates["missing displayName"]})
	if fields, _ := missing["details"].(map[string]any)["fields"].(map[string]any); fields == nil || fields["displayName"] == nil {
		t.Fatalf("validation details must name the field like REST: %v", missing)
	}
	updates := map[string]map[string]any{
		"long displayName": {"displayName": strings.Repeat("я", 256)},
		"wrong managedBy":  {"managedBy": "robot"},
		"unknown field":    {"legacy": true},
	}
	for name, changes := range updates {
		invalid, isError := call(t, session, "endge_update_document", map[string]any{"collection": "queries", "identity": "flights", "revision": 1, "changes": changes})
		if !isError || invalid["error"] != "validation_error" {
			t.Fatalf("update %s must be a REST validation_error: %v", name, invalid)
		}
	}
	if len(h.documents.marks) != 0 {
		t.Fatalf("invalid input must not reach the use case: %v", h.documents.marks)
	}
}

func TestMCPUserSessionRequiresWorkspace(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "jwt")
	// Пользователь без роли Admin ни в одном workspace не видит endge_commit.
	if names := toolNames(t, session); len(names) != 15 || slices.Contains(names, "endge_commit") {
		t.Fatalf("user session without Admin must see all tools except endge_commit, got %v", names)
	}
	missing, isError := call(t, session, "endge_list_documents", map[string]any{"collection": "queries"})
	if !isError || missing["error"] != "workspace_required" || missing["status"] != float64(400) {
		t.Fatalf("JWT without workspace: %v", missing)
	}
	workspaces, _ := call(t, session, "endge_list_workspaces", nil)
	if len(workspaces["workspaces"].([]any)) != 2 {
		t.Fatalf("list workspaces: %v", workspaces)
	}
	updated, isError := call(t, session, "endge_update_document", map[string]any{"collection": "queries", "identity": "flights", "revision": 1, "workspace": "afl", "changes": map[string]any{"displayName": "Y"}})
	if isError || updated["revision"] != float64(2) || h.documents.marks[0] != "user" {
		t.Fatalf("user update: %v %v", updated, h.documents.marks)
	}
	unsupported, isError := call(t, session, "endge_get_document", map[string]any{"collection": "pages", "identity": "x", "workspace": "afl"})
	if !isError || !strings.Contains(unsupported["text"].(string), "enum") {
		t.Fatalf("collection outside the enum must be rejected: %v", unsupported)
	}
}

func TestMCPUserSessionSeesCommitOnlyWithAdminRole(t *testing.T) {
	h := newHarness(t)
	editor := h.connect(t, "jwt")
	if _, err := editor.CallTool(context.Background(), &mcp.CallToolParams{Name: "endge_commit", Arguments: map[string]any{"workspace": "afl", "message": "m", "expectedHeadSequence": 7}}); err == nil {
		t.Fatalf("hidden endge_commit must not be callable by a JWT session without Admin")
	}
	if len(h.commits.created) != 0 {
		t.Fatalf("commit must not reach the use case: %v", h.commits.created)
	}
	for _, principal := range []string{"jwt-admin", "jwt-platform"} {
		if names := toolNames(t, h.connect(t, principal)); len(names) != 16 || !slices.Contains(names, "endge_commit") {
			t.Fatalf("%s must see endge_commit: %v", principal, names)
		}
	}
}
