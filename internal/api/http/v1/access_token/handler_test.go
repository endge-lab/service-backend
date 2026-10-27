package access_token

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_tokens"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// fakeUseCase запоминает вызовы; HTTP-слой проверяется без БД.
type fakeUseCase struct {
	created  []resourceusecase.CreateInput
	listAll  []bool
	revoked  []string
	createFn func(resourceusecase.CreateInput) (*resourceusecase.Created, error)
}

func (f *fakeUseCase) Create(_ context.Context, input resourceusecase.CreateInput) (*resourceusecase.Created, error) {
	f.created = append(f.created, input)
	return f.createFn(input)
}

func (f *fakeUseCase) List(_ context.Context, all bool) ([]entities.AccessToken, error) {
	f.listAll = append(f.listAll, all)
	return []entities.AccessToken{{ID: "t-1", Name: "Codex", Scope: "read", TokenPrefix: "endge_pat_Ab", ExpiresAt: time.Now().Add(time.Hour)}}, nil
}

func (f *fakeUseCase) Revoke(_ context.Context, id string) error {
	f.revoked = append(f.revoked, id)
	if id == "missing" {
		return domainerrors.NotFound("access_token_not_found", "Access token not found")
	}
	return nil
}

func (f *fakeUseCase) MaxTTL() time.Duration { return 720 * time.Hour }

func newTestApp(useCase *fakeUseCase) *fiber.App {
	app := fiber.New()
	RegisterRoutes(app, NewHandler(useCase, appvalidator.NewValidator()))
	return app
}

func decodeBody(t *testing.T, app *fiber.App, method, target, body string, wantStatus int) map[string]any {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d", method, target, response.StatusCode, wantStatus)
	}
	value := map[string]any{}
	if response.ContentLength != 0 {
		_ = json.NewDecoder(response.Body).Decode(&value)
	}
	value["__cacheControl"] = response.Header.Get(fiber.HeaderCacheControl)
	return value
}

// TestListRejectsUnknownOwner проверяет код query_owner_invalid и то, что use case не вызывается.
func TestListRejectsUnknownOwner(t *testing.T) {
	useCase := &fakeUseCase{}
	app := newTestApp(useCase)
	body := decodeBody(t, app, fiber.MethodGet, "/access-tokens?owner=others", "", fiber.StatusBadRequest)
	if body["code"] != "query_owner_invalid" {
		t.Fatalf("code = %v", body["code"])
	}
	if len(useCase.listAll) != 0 {
		t.Fatalf("use case must not be called: %v", useCase.listAll)
	}
	listed := decodeBody(t, app, fiber.MethodGet, "/access-tokens?owner=all", "", fiber.StatusOK)
	if listed["maxTtlSeconds"] != float64(720*3600) || len(listed["items"].([]any)) != 1 {
		t.Fatalf("list response = %v", listed)
	}
	decodeBody(t, app, fiber.MethodGet, "/access-tokens", "", fiber.StatusOK)
	if len(useCase.listAll) != 2 || !useCase.listAll[0] || useCase.listAll[1] {
		t.Fatalf("owner=all → true, default → false: %v", useCase.listAll)
	}
}

// TestCreateReturnsSecretOnceWithNoStore проверяет 201, секрет в ответе и Cache-Control: no-store.
func TestCreateReturnsSecretOnceWithNoStore(t *testing.T) {
	useCase := &fakeUseCase{createFn: func(input resourceusecase.CreateInput) (*resourceusecase.Created, error) {
		return &resourceusecase.Created{Secret: "endge_pat_secret", Token: entities.AccessToken{ID: "t-1", Name: input.Name, Scope: input.Scope, CanCommit: input.CanCommit, ExpiresAt: input.ExpiresAt}}, nil
	}}
	app := newTestApp(useCase)
	body := decodeBody(t, app, fiber.MethodPost, "/access-tokens", `{"workspaceIdentity":"afl","name":"Codex","scope":"write","commit":true,"expiresAt":"2099-01-01T00:00:00Z"}`, fiber.StatusCreated)
	if body["token"] != "endge_pat_secret" || body["__cacheControl"] != "no-store" {
		t.Fatalf("create response = %v", body)
	}
	if len(useCase.created) != 1 || !useCase.created[0].CanCommit || useCase.created[0].Scope != "write" {
		t.Fatalf("create input = %+v", useCase.created)
	}
}

// TestCreateValidatesBodyAndMapsDomainErrors проверяет DTO-валидацию и код ошибки use case.
func TestCreateValidatesBodyAndMapsDomainErrors(t *testing.T) {
	useCase := &fakeUseCase{createFn: func(resourceusecase.CreateInput) (*resourceusecase.Created, error) {
		return nil, domainerrors.Forbidden("access_token_owner_admin_required", "Access token owner must be Workspace Admin to commit")
	}}
	app := newTestApp(useCase)
	invalid := decodeBody(t, app, fiber.MethodPost, "/access-tokens", `{"workspaceIdentity":"afl","name":"Codex","scope":"admin","expiresAt":"2099-01-01T00:00:00Z"}`, fiber.StatusBadRequest)
	if invalid["code"] != "validation_error" || len(useCase.created) != 0 {
		t.Fatalf("invalid scope must fail DTO validation: %v", invalid)
	}
	forbidden := decodeBody(t, app, fiber.MethodPost, "/access-tokens", `{"workspaceIdentity":"afl","name":"Codex","scope":"write","commit":true,"expiresAt":"2099-01-01T00:00:00Z"}`, fiber.StatusForbidden)
	if forbidden["code"] != "access_token_owner_admin_required" {
		t.Fatalf("domain error code = %v", forbidden)
	}
}

// TestRevokeReturnsNoContentAndNotFound проверяет 204 и 404 access_token_not_found.
func TestRevokeReturnsNoContentAndNotFound(t *testing.T) {
	useCase := &fakeUseCase{}
	app := newTestApp(useCase)
	decodeBody(t, app, fiber.MethodDelete, "/access-tokens/t-1", "", fiber.StatusNoContent)
	missing := decodeBody(t, app, fiber.MethodDelete, "/access-tokens/missing", "", fiber.StatusNotFound)
	if missing["code"] != "access_token_not_found" {
		t.Fatalf("revoke missing = %v", missing)
	}
}
