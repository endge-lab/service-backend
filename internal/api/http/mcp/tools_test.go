package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	validator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func newTestTools() (*tools, *fakeDocuments) {
	useCases, documents, _, _ := newFakeUseCases()
	return &tools{cfg: Config{Version: "0.16.0"}, useCases: useCases, contracts: BindContracts(validator.NewValidator()), logger: zap.NewNop()}, documents
}

func decodeResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	value := map[string]any{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPermissionsFollowAccessToken(t *testing.T) {
	cases := map[string]struct {
		principal *entities.AccessTokenPrincipal
		want      permissions
	}{
		"user session": {nil, permissions{write: true, commit: true, delete: true}},
		"read":         {&entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeRead, CanCommit: true, CanDelete: true}, permissions{}},
		"write":        {&entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite}, permissions{write: true}},
		"write+delete": {&entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite, CanDelete: true}, permissions{write: true, delete: true}},
	}
	for name, tc := range cases {
		ctx := context.Background()
		if tc.principal != nil {
			ctx = entities.WithAccessToken(ctx, *tc.principal)
		}
		if got := permissionsFrom(ctx); got != tc.want {
			t.Fatalf("%s: got %+v, want %+v", name, got, tc.want)
		}
	}
}

// Инструменты скрыты от токена без флагов, но прямой вызов обработчика тоже отклоняется.
func TestDeleteAndCommitCheckTokenFlags(t *testing.T) {
	tools, documents := newTestTools()
	ctx := entities.WithAccessToken(context.Background(), entities.AccessTokenPrincipal{TokenID: "t", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeWrite})
	result, _, err := tools.deleteDocument(ctx, nil, mutateInput{Collection: "queries", Identity: "flights", Revision: 1})
	if err != nil || !result.IsError || decodeResult(t, result)["error"] != "access_token_delete_forbidden" || len(documents.marks) != 0 {
		t.Fatalf("delete without flag: %v %v", result, err)
	}
	result, _, err = tools.commit(ctx, nil, commitInput{Message: "x", ExpectedHeadSequence: 7})
	if err != nil || !result.IsError || decodeResult(t, result)["error"] != "access_token_commit_forbidden" {
		t.Fatalf("commit without flag: %v %v", result, err)
	}
}

func TestReadTokenWriteIsRejectedByUseCase(t *testing.T) {
	tools, _ := newTestTools()
	ctx := entities.WithAccessToken(context.Background(), entities.AccessTokenPrincipal{TokenID: "t", WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeRead})
	result, _, _ := tools.updateDocument(ctx, nil, updateInput{Collection: "queries", Identity: "flights", Revision: 1, Changes: map[string]any{"displayName": "X"}})
	body := decodeResult(t, result)
	if !result.IsError || body["error"] != "access_token_scope_insufficient" || !strings.Contains(body["hint"].(string), "только для чтения") {
		t.Fatalf("read token update: %v", body)
	}
}

func TestErrorResultHidesInternalCause(t *testing.T) {
	result, _, err := errorResult(zap.NewNop(), "endge_status", errors.New("pq: password=secret"))
	body := decodeResult(t, result)
	if err != nil || !result.IsError || body["error"] != "internal_error" || strings.Contains(result.Content[0].(*mcp.TextContent).Text, "secret") {
		t.Fatalf("internal error must be safe: %v", body)
	}
}

func TestUnknownCollectionUsesBackendCode(t *testing.T) {
	tools, _ := newTestTools()
	ctx := entities.WithAccessToken(context.Background(), entities.AccessTokenPrincipal{WorkspaceIdentity: "afl", Scope: entities.AccessTokenScopeRead})
	result, _, _ := tools.getDocument(ctx, nil, documentInput{Collection: "stores", Identity: "x"})
	if body := decodeResult(t, result); body["error"] != "collection_unsupported" || body["status"] != float64(400) {
		t.Fatalf("unknown collection: %v", body)
	}
}

func TestSummarizeMergesSnapshotState(t *testing.T) {
	got := summarize(map[string]any{"identity": "q", "displayName": "Q", "source": "big", "description": "", "state": map[string]any{"revision": 4.0, "updatedAt": "t"}})
	if got["revision"] != 4.0 || got["updatedAt"] != "t" || got["source"] != nil || got["description"] != nil {
		t.Fatalf("summary: %v", got)
	}
}

func TestInstructionsWithoutDocsURL(t *testing.T) {
	if text := instructions(Config{}); strings.Contains(text, "llms") || !strings.Contains(text, "endge_status") {
		t.Fatalf("instructions without docs: %s", text)
	}
	if docLinks("") != nil || len(docLinks("https://d/")) != 4 || docLinks("https://d/")[0]["url"] != "https://d/llms.txt" {
		t.Fatalf("doc links")
	}
}

// TestErrorResultCarriesCodeStatusAndHint проверяет, что каждая ошибка с подсказкой
// доходит до агента с кодом backend, HTTP-статусом и непустым hint.
func TestErrorResultCarriesCodeStatusAndHint(t *testing.T) {
	cases := []struct {
		err    error
		status float64
	}{
		{domainerrors.Conflict("revision_conflict", "Revision conflict"), 409},
		{domainerrors.InvalidInput("expected_revision_invalid", "revision required"), 400},
		{domainerrors.Conflict("head_sequence_conflict", "Workspace head changed"), 409},
		{domainerrors.InvalidInput("workspace_required", "workspace required"), 400},
		{domainerrors.Forbidden("access_token_workspace_forbidden", "other workspace"), 403},
		{domainerrors.Forbidden("access_token_scope_insufficient", "read-only"), 403},
		{domainerrors.Forbidden("access_token_delete_forbidden", "no delete"), 403},
		{domainerrors.Forbidden("access_token_commit_forbidden", "no commit"), 403},
		{domainerrors.Forbidden("access_token_owner_admin_required", "owner must be admin"), 403},
		{domainerrors.InvalidInput("validation_error", "invalid"), 400},
		{domainerrors.Forbidden("access_token_admin_forbidden", "admin only"), 403},
		{domainerrors.InvalidInput("collection_unsupported", "unknown collection"), 400},
		{domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required"), 403},
		{domainerrors.NotFound("document_not_found", "Document not found"), 404},
	}
	for _, tc := range cases {
		code := domainerrors.CodeOf(tc.err)
		result, _, err := errorResult(zap.NewNop(), "endge_commit", tc.err)
		body := decodeResult(t, result)
		if err != nil || !result.IsError || body["error"] != code || body["status"] != tc.status {
			t.Fatalf("%s: unexpected result %v", code, body)
		}
		if hint, _ := body["hint"].(string); hint == "" {
			t.Fatalf("%s: hint must not be empty: %v", code, body)
		}
	}
	// Ошибка без отдельной подсказки и без типового статуса уходит без hint.
	result, _, _ := errorResult(zap.NewNop(), "endge_status", domainerrors.Conflict("other_conflict", "x"))
	if body := decodeResult(t, result); body["hint"] != nil {
		t.Fatalf("unexpected hint: %v", body)
	}
}
