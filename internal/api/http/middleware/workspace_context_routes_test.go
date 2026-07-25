package middleware_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/api/http/health"
	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/openapi"
	workspacehttp "github.com/endge-lab/service-backend/internal/api/http/v1/workspace"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

type workspaceRoutesStub struct{}

type workspaceResolverStub struct{}

func (workspaceResolverStub) GetByIdentity(context.Context, string) (*entities.RWorkspace, error) {
	return nil, nil
}

func (workspaceRoutesStub) Create(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) }
func (workspaceRoutesStub) List(c *fiber.Ctx) error   { return c.SendStatus(fiber.StatusNoContent) }
func (workspaceRoutesStub) Get(c *fiber.Ctx) error    { return c.SendStatus(fiber.StatusNoContent) }
func (workspaceRoutesStub) Update(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) }

func TestWorkspaceContextAppliesOnlyToScopedRoutes(t *testing.T) {
	app := newWorkspaceContextTestApp()
	app.Get("/api/v1/projects", httpmiddleware.NewWorkspaceContextMiddleware(workspaceResolverStub{}).RequireWorkspace(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	for _, path := range []string{"/health", "/swagger/openapi3.yaml", "/api/v1/workspaces/"} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != fiber.StatusOK && response.StatusCode != fiber.StatusNoContent {
			t.Errorf("GET %s without workspace header = %d, want successful response", path, response.StatusCode)
		}
	}

	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/projects", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("scoped route status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
}

func TestCORSPreflightAllowsWorkspaceHeader(t *testing.T) {
	app := newWorkspaceContextTestApp()
	request := httptest.NewRequest(fiber.MethodOptions, "/api/v1/projects", nil)
	request.Header.Set(fiber.HeaderOrigin, "https://frontend.example")
	request.Header.Set(fiber.HeaderAccessControlRequestMethod, fiber.MethodGet)
	request.Header.Set(fiber.HeaderAccessControlRequestHeaders, httpmiddleware.WorkspaceHeader)

	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.StatusCode, fiber.StatusNoContent)
	}
	if !strings.Contains(strings.ToLower(response.Header.Get(fiber.HeaderAccessControlAllowHeaders)), strings.ToLower(httpmiddleware.WorkspaceHeader)) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %q", response.Header.Get(fiber.HeaderAccessControlAllowHeaders), httpmiddleware.WorkspaceHeader)
	}
}

func newWorkspaceContextTestApp() *fiber.App {
	app := fiber.New()
	httpmiddleware.Register(app, &config.Config{HTTP: config.HTTPConfig{CORSAllowedOrigins: "https://frontend.example"}}, noop.NewMeterProvider().Meter("workspace-context-test"), zap.NewNop())
	health.RegisterRoutes(app, health.Config{})
	openapi.RegisterRoutes(app)
	workspacehttp.RegisterRoutes(app.Group("/api"), workspaceRoutesStub{})
	return app
}
