package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type workspaceServiceStub struct {
	value       *entities.RWorkspace
	createInput *workspaces.CreateWorkspaceInput
	updateInput *workspaces.UpdateWorkspaceInput
}

var _ UseCase = (*workspaceServiceStub)(nil)

func (s *workspaceServiceStub) Create(_ context.Context, input workspaces.CreateWorkspaceInput) (*entities.RWorkspace, error) {
	s.createInput = &input
	return s.value, nil
}

func (s *workspaceServiceStub) List(context.Context) ([]*entities.RWorkspace, error) {
	return []*entities.RWorkspace{s.value}, nil
}

func (s *workspaceServiceStub) GetByIdentity(context.Context, string) (*entities.RWorkspace, error) {
	return s.value, nil
}

func (s *workspaceServiceStub) Update(_ context.Context, input workspaces.UpdateWorkspaceInput) (*entities.RWorkspace, error) {
	s.updateInput = &input
	return s.value, nil
}

func TestWorkspaceHandlersCRUD(t *testing.T) {
	manualToken := "must-not-be-exposed"
	service := &workspaceServiceStub{value: &entities.RWorkspace{
		ID:          uuid.New(),
		Identity:    "default",
		DisplayName: "Default workspace",
		Configuration: entities.EndgeConfiguration{
			SSE: &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeManual, ManualToken: &manualToken},
		},
		CreatedAt: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
	}}
	handler := NewHandler(service, appvalidator.NewValidator(), zap.NewNop(), otel.Tracer("test"))
	app := fiber.New()
	routes := app.Group("/api/v1/workspaces")
	routes.Post("/", handler.Create)
	routes.Get("/", handler.List)
	routes.Get("/:workspace_identity", handler.Get)
	routes.Patch("/:workspace_identity", handler.Update)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "create", method: http.MethodPost, path: "/api/v1/workspaces/", body: `{"identity":"default","displayName":"Default workspace"}`, wantStatus: http.StatusCreated},
		{name: "list", method: http.MethodGet, path: "/api/v1/workspaces/", wantStatus: http.StatusOK},
		{name: "get", method: http.MethodGet, path: "/api/v1/workspaces/default", wantStatus: http.StatusOK},
		{name: "update", method: http.MethodPatch, path: "/api/v1/workspaces/default", body: `{"displayName":"Renamed workspace"}`, wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			if test.body != "" {
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			}
			request.Host = "service-backend.test"
			result, err := app.Test(request)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer result.Body.Close()
			if result.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", result.StatusCode, test.wantStatus)
			}
			body, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if bytes.Contains(body, []byte(manualToken)) {
				t.Fatal("response exposes sse.manualToken")
			}
		})
	}

	if service.createInput == nil || service.createInput.Identity != "default" || service.createInput.Configuration != nil {
		t.Fatalf("unexpected create input: %+v", service.createInput)
	}
	if service.updateInput == nil || service.updateInput.Identity != "default" || service.updateInput.DisplayName == nil || *service.updateInput.DisplayName != "Renamed workspace" {
		t.Fatalf("unexpected update input: %+v", service.updateInput)
	}
}

func TestWorkspaceCreateRejectsInvalidRequest(t *testing.T) {
	service := &workspaceServiceStub{value: &entities.RWorkspace{}}
	handler := NewHandler(service, appvalidator.NewValidator(), zap.NewNop(), otel.Tracer("test"))
	app := fiber.New()
	app.Post("/api/v1/workspaces", handler.Create)

	request, err := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(`{"identity":""}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Host = "service-backend.test"
	result, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Body.Close()
	if result.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", result.StatusCode, http.StatusBadRequest)
	}
	var response struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(result.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "validation_error" {
		t.Fatalf("error code = %q, want validation_error", response.Code)
	}
}
