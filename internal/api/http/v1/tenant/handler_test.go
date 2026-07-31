package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/tenants"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestTenantHandlersPassPublicInputsAndPreservePatchNulls(t *testing.T) {
	manualToken := "must-not-be-exposed"
	service := &tenantServiceStub{value: tenantView(manualToken)}
	handler := NewHandler(service, appvalidator.NewValidator(), observability.NewCore(otel.Tracer("test"), zap.NewNop()), nil)
	app := fiber.New()
	routes := app.Group("/api/v1/tenants")
	routes.Post("/", handler.CreateTenant)
	routes.Get("/", handler.ListTenants)
	routes.Get("/:tenant_identity", handler.GetTenantByIdentity)
	routes.Patch("/:tenant_identity", handler.UpdateTenant)
	routes.Delete("/:tenant_identity", handler.HardDeleteTenant)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "create", method: http.MethodPost, path: "/api/v1/tenants/", body: `{"identity":"tenant-default","displayName":"Default tenant","code":"TENANT_DEFAULT","folderIdentity":"root-tenants","configuration":{"mode":"inherit","patch":{}}}`, wantStatus: http.StatusCreated},
		{name: "create with default configuration", method: http.MethodPost, path: "/api/v1/tenants/", body: `{"identity":"tenant-default","displayName":"Default tenant","code":"TENANT_DEFAULT"}`, wantStatus: http.StatusCreated},
		{name: "list", method: http.MethodGet, path: "/api/v1/tenants/?folder_identity=root-tenants", wantStatus: http.StatusOK},
		{name: "get", method: http.MethodGet, path: "/api/v1/tenants/tenant-default", wantStatus: http.StatusOK},
		{name: "patch nulls", method: http.MethodPatch, path: "/api/v1/tenants/tenant-default", body: `{"displayName":"Renamed tenant","description":null,"folderIdentity":null}`, wantStatus: http.StatusOK},
		{name: "delete", method: http.MethodDelete, path: "/api/v1/tenants/tenant-default", wantStatus: http.StatusNoContent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Host = "service-backend.test"
			if test.body != "" {
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(body, []byte(manualToken)) {
				t.Fatal("tenant response exposes sse.manualToken")
			}
		})
	}

	if len(service.createInputs) != 2 || service.createInputs[0].FolderIdentity == nil || *service.createInputs[0].FolderIdentity != "root-tenants" || service.createInputs[0].Configuration == nil || service.createInputs[1].Configuration != nil {
		t.Fatalf("unexpected create inputs: %+v", service.createInputs)
	}
	if service.listInput == nil || service.listInput.FolderIdentity == nil || *service.listInput.FolderIdentity != "root-tenants" {
		t.Fatalf("unexpected list input: %+v", service.listInput)
	}
	if service.updateInput == nil || service.updateInput.DisplayName == nil || *service.updateInput.DisplayName != "Renamed tenant" || !service.updateInput.Description.Set || service.updateInput.Description.Value != nil || !service.updateInput.FolderIdentity.Set || service.updateInput.FolderIdentity.Value != nil || service.updateInput.ConfigurationSet {
		t.Fatalf("unexpected update input: %+v", service.updateInput)
	}
	if service.deletedIdentity != "tenant-default" {
		t.Fatalf("deleted identity = %q", service.deletedIdentity)
	}
}

func TestTenantRoutesRequireWorkspaceHeader(t *testing.T) {
	service := &tenantServiceStub{value: tenantView("")}
	handler := NewHandler(service, appvalidator.NewValidator(), observability.NewCore(otel.Tracer("test"), zap.NewNop()), nil)
	app := fiber.New()
	RegisterRoutes(app.Group("/api"), handler, middleware.NewWorkspaceContextMiddleware(workspaceResolverStub{}))

	request, err := http.NewRequest(http.MethodGet, "/api/v1/tenants/", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "service-backend.test"
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

type tenantServiceStub struct {
	value           *tenants.TenantView
	createInput     *tenants.CreateTenantInput
	createInputs    []*tenants.CreateTenantInput
	listInput       *tenants.ListTenantsInput
	updateInput     *tenants.UpdateTenantInput
	deletedIdentity string
}

var _ UseCase = (*tenantServiceStub)(nil)

func (s *tenantServiceStub) Create(_ context.Context, input tenants.CreateTenantInput) (*tenants.TenantView, error) {
	s.createInput = &input
	s.createInputs = append(s.createInputs, &input)
	return s.value, nil
}
func (s *tenantServiceStub) List(_ context.Context, input tenants.ListTenantsInput) ([]*tenants.TenantView, error) {
	s.listInput = &input
	return []*tenants.TenantView{s.value}, nil
}
func (s *tenantServiceStub) GetByIdentity(context.Context, string) (*tenants.TenantView, error) {
	return s.value, nil
}
func (s *tenantServiceStub) Update(_ context.Context, input tenants.UpdateTenantInput) (*tenants.TenantView, error) {
	s.updateInput = &input
	return s.value, nil
}
func (s *tenantServiceStub) HardDelete(_ context.Context, identity string) error {
	s.deletedIdentity = identity
	return nil
}

type workspaceResolverStub struct{}

func (workspaceResolverStub) GetByIdentity(context.Context, string) (*entities.RWorkspace, error) {
	return &entities.RWorkspace{ID: uuid.New(), Identity: "demo-workspace"}, nil
}

func tenantView(manualToken string) *tenants.TenantView {
	folderIdentity := "root-tenants"
	configuration := entities.DefaultEndgeConfiguration()
	configuration.SSE = &entities.EndgeSSEConfiguration{AuthMode: entities.SSEAuthModeManual, ManualToken: &manualToken}
	return &tenants.TenantView{
		Tenant:         &entities.RTenant{ID: uuid.New(), Identity: "tenant-default", DisplayName: "Default tenant", Code: "TENANT_DEFAULT", Configuration: entities.EndgeConfigurationContribution{Mode: entities.EndgeConfigurationContributionModeReplace, Patch: map[string]json.RawMessage{}, Value: &configuration}, CreatedAt: time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)},
		FolderIdentity: &folderIdentity,
	}
}
