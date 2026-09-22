//go:build e2e

package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type aiAPIFixture struct {
	app       *fiber.App
	database  *support.TestDatabase
	workbench *support.FakeWorkbench
	platform  map[string]string
	viewer    map[string]string
	outsider  map[string]string
	workspace string
}

// TestAICatalogHTTPContracts verifies every AI catalog endpoint, including
// connection/model lifecycle, RBAC and encrypted credential storage.
func TestAICatalogHTTPContracts(t *testing.T) {
	fixture := newAIAPIFixture(t)
	adapters := "/api/v1/ai/provider-adapters"
	connections := "/api/v1/ai/provider-connections"
	models := "/api/v1/ai/model-profiles"

	unauthenticatedAdapters := perform(t, fixture.app, http.MethodGet, adapters, nil, nil)
	assertStatus(t, unauthenticatedAdapters, fiber.StatusUnauthorized)
	unauthenticatedAdapters.Body.Close()
	viewerAdapters := perform(t, fixture.app, http.MethodGet, adapters, nil, fixture.viewer)
	assertStatus(t, viewerAdapters, fiber.StatusOK)
	if len(listItems(t, decodeObject(t, viewerAdapters))) != 2 {
		t.Fatal("AI adapter catalog does not expose both supported adapters")
	}

	unauthenticatedConnections := perform(t, fixture.app, http.MethodGet, connections, nil, nil)
	assertStatus(t, unauthenticatedConnections, fiber.StatusUnauthorized)
	unauthenticatedConnections.Body.Close()
	viewerPrivate := perform(t, fixture.app, http.MethodPost, connections, map[string]any{
		"name": "Viewer private", "adapter": "ollama", "baseUrl": "http://ollama.test", "visibility": "private", "enabled": true,
	}, fixture.viewer)
	assertStatus(t, viewerPrivate, fiber.StatusCreated)
	viewerPrivate.Body.Close()
	viewerPublic := perform(t, fixture.app, http.MethodPost, connections, map[string]any{
		"name": "Denied public", "adapter": "ollama", "baseUrl": "http://ollama.test", "visibility": "public", "enabled": true,
	}, fixture.viewer)
	assertStatus(t, viewerPublic, fiber.StatusForbidden)
	viewerPublic.Body.Close()

	secret := "catalog-secret-before-replace"
	createdConnection := perform(t, fixture.app, http.MethodPost, connections, map[string]any{
		"name": "Platform connection", "adapter": "anthropic", "baseUrl": "https://api.example.test/", "credential": secret, "visibility": "public", "enabled": true,
	}, fixture.platform)
	assertStatus(t, createdConnection, fiber.StatusCreated)
	createdConnectionBody := decodeObject(t, createdConnection)
	connectionID := stringField(t, createdConnectionBody, "id")
	if createdConnectionBody["hasCredential"] != true || strings.Contains(stringifyJSON(t, createdConnectionBody), secret) {
		t.Fatalf("connection response leaks or misses credential state: %#v", createdConnectionBody)
	}
	assertCredentialEncrypted(t, fixture.database, connectionID, secret)

	patchedConnection := perform(t, fixture.app, http.MethodPatch, connections+"/"+connectionID, map[string]any{"name": "Patched platform connection"}, fixture.platform)
	assertStatus(t, patchedConnection, fiber.StatusOK)
	if stringField(t, decodeObject(t, patchedConnection), "name") != "Patched platform connection" {
		t.Fatal("connection patch did not persist its name")
	}
	replacedSecret := "catalog-secret-after-replace"
	replacedCredential := perform(t, fixture.app, http.MethodPut, connections+"/"+connectionID+"/credential", map[string]any{"credential": replacedSecret}, fixture.platform)
	assertStatus(t, replacedCredential, fiber.StatusOK)
	if strings.Contains(stringifyJSON(t, decodeObject(t, replacedCredential)), replacedSecret) {
		t.Fatal("credential replacement leaked plaintext secret")
	}
	assertCredentialEncrypted(t, fixture.database, connectionID, replacedSecret)

	createdWithModel := perform(t, fixture.app, http.MethodPost, connections+"/with-model", aiConnectionWithModelPayload("Catalog with model", "catalog-model", "catalog-secret"), fixture.platform)
	assertStatus(t, createdWithModel, fiber.StatusCreated)
	withModelBody := decodeObject(t, createdWithModel)
	modelID := stringField(t, objectField(t, withModelBody, "model"), "id")
	withModelConnectionID := stringField(t, objectField(t, withModelBody, "connection"), "id")

	unauthenticatedModels := perform(t, fixture.app, http.MethodGet, models, nil, nil)
	assertStatus(t, unauthenticatedModels, fiber.StatusUnauthorized)
	unauthenticatedModels.Body.Close()
	modelList := perform(t, fixture.app, http.MethodGet, models, nil, fixture.viewer)
	assertStatus(t, modelList, fiber.StatusOK)
	if !hasID(t, listItems(t, decodeObject(t, modelList)), modelID) {
		t.Fatal("public model is absent from viewer model catalog")
	}
	viewerModel := perform(t, fixture.app, http.MethodPost, models, map[string]any{
		"connectionId": withModelConnectionID, "providerModelId": "viewer-denied", "displayName": "Viewer denied", "enabled": true,
	}, fixture.viewer)
	assertStatus(t, viewerModel, fiber.StatusForbidden)
	viewerModel.Body.Close()
	createdModel := perform(t, fixture.app, http.MethodPost, models, map[string]any{
		"connectionId": withModelConnectionID, "providerModelId": "catalog-extra", "displayName": "Catalog extra", "enabled": true,
	}, fixture.platform)
	assertStatus(t, createdModel, fiber.StatusCreated)
	extraModelID := stringField(t, decodeObject(t, createdModel), "id")
	patchedModel := perform(t, fixture.app, http.MethodPatch, models+"/"+extraModelID, map[string]any{"displayName": "Catalog extra patched"}, fixture.platform)
	assertStatus(t, patchedModel, fiber.StatusOK)
	if stringField(t, decodeObject(t, patchedModel), "displayName") != "Catalog extra patched" {
		t.Fatal("model patch did not persist displayName")
	}
	deletedModel := perform(t, fixture.app, http.MethodDelete, models+"/"+extraModelID, nil, fixture.platform)
	assertStatus(t, deletedModel, fiber.StatusNoContent)
	deletedModel.Body.Close()
	deletedConnection := perform(t, fixture.app, http.MethodDelete, connections+"/"+withModelConnectionID, nil, fixture.platform)
	assertStatus(t, deletedConnection, fiber.StatusNoContent)
	deletedConnection.Body.Close()
}

// TestAIConversationHTTPContracts verifies capabilities and the complete
// workspace-scoped conversation lifecycle through the actual gRPC adapter.
func TestAIConversationHTTPContracts(t *testing.T) {
	fixture := newAIAPIFixture(t)
	connectionWithModel := perform(t, fixture.app, http.MethodPost, "/api/v1/ai/provider-connections/with-model", aiConnectionWithModelPayload("Conversation connection", "conversation-model", "run-secret"), fixture.platform)
	assertStatus(t, connectionWithModel, fiber.StatusCreated)
	modelID := stringField(t, objectField(t, decodeObject(t, connectionWithModel), "model"), "id")

	unauthenticatedCapabilities := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/capabilities", nil, map[string]string{"X-Endge-Workspace": fixture.workspace})
	assertStatus(t, unauthenticatedCapabilities, fiber.StatusUnauthorized)
	unauthenticatedCapabilities.Body.Close()
	capabilities := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/capabilities", nil, fixture.viewer)
	assertStatus(t, capabilities, fiber.StatusOK)
	capabilitiesBody := decodeObject(t, capabilities)
	adapters, _ := capabilitiesBody["adapters"].([]any)
	if capabilitiesBody["available"] != true || capabilitiesBody["canRun"] != true || len(adapters) != 2 {
		t.Fatalf("unexpected AI capabilities: %#v", capabilitiesBody)
	}
	unauthenticatedList := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/conversations", nil, map[string]string{"X-Endge-Workspace": fixture.workspace})
	assertStatus(t, unauthenticatedList, fiber.StatusUnauthorized)
	unauthenticatedList.Body.Close()
	outsiderCapabilities := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/capabilities", nil, fixture.outsider)
	assertStatus(t, outsiderCapabilities, fiber.StatusForbidden)
	outsiderCapabilities.Body.Close()

	created := perform(t, fixture.app, http.MethodPost, "/api/v1/ai/conversations", map[string]any{"modelProfileId": modelID}, fixture.viewer)
	assertStatus(t, created, fiber.StatusCreated)
	conversationID := stringField(t, decodeObject(t, created), "id")
	listed := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/conversations", nil, fixture.viewer)
	assertStatus(t, listed, fiber.StatusOK)
	if !hasID(t, listItems(t, decodeObject(t, listed)), conversationID) {
		t.Fatal("created conversation is absent from its list")
	}
	patched := perform(t, fixture.app, http.MethodPatch, "/api/v1/ai/conversations/"+conversationID, map[string]any{"modelProfileId": modelID}, fixture.viewer)
	assertStatus(t, patched, fiber.StatusOK)
	patched.Body.Close()

	run := perform(t, fixture.app, http.MethodPost, "/api/v1/ai/conversations/"+conversationID+"/runs", map[string]any{
		"requestId": uuid.NewString(), "modelProfileId": modelID, "prompt": "Explain this workspace",
	}, fixture.viewer)
	assertStatus(t, run, fiber.StatusOK)
	runBody, err := io.ReadAll(run.Body)
	run.Body.Close()
	if err != nil || !strings.Contains(string(runBody), "event: completed") {
		t.Fatalf("AI run SSE=%q err=%v", runBody, err)
	}
	lastRun := fixture.workbench.LastRun()
	if lastRun == nil || lastRun.GetWorkspace().GetId() == "" || lastRun.GetProviderAccess().GetCredential() != "run-secret" {
		t.Fatal("backend did not send the authorized workspace/provider access to Workbench")
	}
	messages := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/conversations/"+conversationID+"/messages", nil, fixture.viewer)
	assertStatus(t, messages, fiber.StatusOK)
	if len(listItems(t, decodeObject(t, messages))) != 1 {
		t.Fatal("completed fake run did not expose its assistant message")
	}
	reset := perform(t, fixture.app, http.MethodPost, "/api/v1/ai/conversations/reset", map[string]any{"currentConversationId": conversationID, "modelProfileId": modelID}, fixture.viewer)
	assertStatus(t, reset, fiber.StatusCreated)
	if stringField(t, decodeObject(t, reset), "id") == conversationID {
		t.Fatal("conversation reset did not create a new conversation")
	}
	outsiderList := perform(t, fixture.app, http.MethodGet, "/api/v1/ai/conversations", nil, fixture.outsider)
	assertStatus(t, outsiderList, fiber.StatusForbidden)
	outsiderList.Body.Close()
}

func newAIAPIFixture(t *testing.T) *aiAPIFixture {
	t.Helper()
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	workbench := support.NewFakeWorkbench(t)
	cfg := support.OIDCConfig(provider)
	cfg.AIWorkbench.GRPCTarget = workbench.Target()
	app := support.NewTestApp(t, database, cfg)
	platform := bearer(provider.Token(t, support.TokenInput{Subject: "ai-platform", Username: "ai-platform", DisplayName: "AI Platform", Groups: []string{"endge-platform-admins"}}))
	viewer := bearer(provider.Token(t, support.TokenInput{Subject: "ai-viewer", Username: "ai-viewer", DisplayName: "AI Viewer"}))
	outsider := bearer(provider.Token(t, support.TokenInput{Subject: "ai-outsider", Username: "ai-outsider", DisplayName: "AI Outsider"}))
	_ = currentUserID(t, app, platform)
	viewerID := currentUserID(t, app, viewer)
	_ = currentUserID(t, app, outsider)
	workspace := "ai-workspace"
	createWorkspace(t, app, platform, workspace)
	putMembership(t, app, platform, workspace, viewerID, "viewer")
	return &aiAPIFixture{app: app, database: database, workbench: workbench, platform: workspaceHeaders(platform, workspace), viewer: workspaceHeaders(viewer, workspace), outsider: workspaceHeaders(outsider, workspace), workspace: workspace}
}

func aiConnectionWithModelPayload(name, providerModelID, credential string) map[string]any {
	return map[string]any{
		"name": name, "adapter": "anthropic", "baseUrl": "https://api.example.test", "credential": credential, "visibility": "public", "enabled": true,
		"model": map[string]any{"providerModelId": providerModelID, "displayName": providerModelID, "enabled": true, "isDefault": true},
	}
}

func assertCredentialEncrypted(t *testing.T, database *support.TestDatabase, connectionID, plaintext string) {
	t.Helper()
	var encrypted []byte
	if err := database.Pool.QueryRow(t.Context(), `SELECT credential_encrypted FROM ai_provider_connections WHERE id=$1`, connectionID).Scan(&encrypted); err != nil {
		t.Fatalf("load encrypted AI credential: %v", err)
	}
	if len(encrypted) == 0 || string(encrypted) == plaintext || strings.Contains(string(encrypted), plaintext) {
		t.Fatal("AI credential is absent or stored as plaintext")
	}
}

func stringifyJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON response: %v", err)
	}
	return string(encoded)
}
