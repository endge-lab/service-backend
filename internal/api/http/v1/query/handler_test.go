package query

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type useCaseStub struct {
	createCalls int
	patchCalls  int
	expected    int
}

func (s *useCaseStub) List(context.Context, ports.DocumentFilter) ([]entities.Document, error) {
	return nil, nil
}
func (s *useCaseStub) Get(context.Context, string, bool) (*entities.Document, error) {
	return document(1), nil
}
func (s *useCaseStub) Create(context.Context, documents.CreateInput) (*entities.Document, error) {
	s.createCalls++
	return document(1), nil
}
func (s *useCaseStub) Patch(_ context.Context, _ string, _ documents.PatchInput, expected int) (*entities.Document, error) {
	s.patchCalls++
	s.expected = expected
	return document(2), nil
}
func (s *useCaseStub) Delete(context.Context, string, int) (*entities.Document, error) {
	return document(2), nil
}
func (s *useCaseStub) Restore(context.Context, string, int) (*entities.Document, error) {
	return document(2), nil
}

// TestHandlerCreateValidationAndTypedContract проверяет transport validation и типизированный ответ Query.
func TestHandlerCreateValidationAndTypedContract(t *testing.T) {
	t.Parallel()
	stub := &useCaseStub{}
	app := fiber.New()
	RegisterRoutes(app, NewHandler(stub, appvalidator.NewValidator()))

	invalid := httptest.NewRequest(fiber.MethodPost, "/queries/", strings.NewReader(`{"identity":"q","displayName":"Q","source":"query Q","sourceVersion":1}`))
	invalid.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(invalid)
	if err != nil {
		t.Fatalf("invalid request: %v", err)
	}
	if response.StatusCode != fiber.StatusBadRequest || stub.createCalls != 0 {
		t.Fatalf("invalid status = %d, create calls = %d", response.StatusCode, stub.createCalls)
	}
	var validation struct {
		Code    string `json:"code"`
		Details struct {
			Fields map[string]string `json:"fields"`
		} `json:"details"`
	}
	if err = json.NewDecoder(response.Body).Decode(&validation); err != nil {
		t.Fatalf("decode validation response: %v", err)
	}
	if validation.Code != "validation_error" || validation.Details.Fields["sourceVersion"] == "" {
		t.Fatalf("validation response = %#v", validation)
	}

	valid := httptest.NewRequest(fiber.MethodPost, "/queries/", strings.NewReader(`{"identity":"q","displayName":"Q","source":"query Q","sourceVersion":2}`))
	valid.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err = app.Test(valid)
	if err != nil {
		t.Fatalf("valid request: %v", err)
	}
	if response.StatusCode != fiber.StatusCreated || response.Header.Get(fiber.HeaderETag) != `"1"` || stub.createCalls != 1 {
		t.Fatalf("valid status = %d, etag = %q, calls = %d", response.StatusCode, response.Header.Get(fiber.HeaderETag), stub.createCalls)
	}
}

// TestHandlerPatchRequiresAndPassesIfMatch проверяет обязательный optimistic-lock header.
func TestHandlerPatchRequiresAndPassesIfMatch(t *testing.T) {
	t.Parallel()
	stub := &useCaseStub{}
	app := fiber.New()
	RegisterRoutes(app, NewHandler(stub, appvalidator.NewValidator()))
	body := `{"source":"updated","sourceVersion":2}`

	missing := httptest.NewRequest(fiber.MethodPatch, "/queries/q", strings.NewReader(body))
	missing.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(missing)
	if err != nil {
		t.Fatalf("missing If-Match request: %v", err)
	}
	if response.StatusCode != fiber.StatusPreconditionRequired || stub.patchCalls != 0 {
		t.Fatalf("missing status = %d, calls = %d", response.StatusCode, stub.patchCalls)
	}

	valid := httptest.NewRequest(fiber.MethodPatch, "/queries/q", strings.NewReader(body))
	valid.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	valid.Header.Set(fiber.HeaderIfMatch, `"1"`)
	response, err = app.Test(valid)
	if err != nil {
		t.Fatalf("valid patch request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK || response.Header.Get(fiber.HeaderETag) != `"2"` || stub.expected != 1 {
		t.Fatalf("patch status = %d, etag = %q, expected = %d", response.StatusCode, response.Header.Get(fiber.HeaderETag), stub.expected)
	}
}

func document(revision int) *entities.Document {
	return &entities.Document{
		ID: "9d093dad-7682-4ec6-aaba-c0deab2a8f92", Identity: "q", DisplayName: "Q",
		ManagedBy: "user", Meta: json.RawMessage(`{}`), Active: true, Revision: revision,
		Data: json.RawMessage(`{"source":"query Q","sourceVersion":2}`),
	}
}
