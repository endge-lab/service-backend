package middleware

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type workspaceResolverStub struct {
	workspace  *entities.RWorkspace
	err        error
	identities []string
}

func (s *workspaceResolverStub) GetByIdentity(_ context.Context, identity string) (*entities.RWorkspace, error) {
	s.identities = append(s.identities, identity)
	return s.workspace, s.err
}

func TestRequireWorkspaceMapsNotFound(t *testing.T) {
	resolver := &workspaceResolverStub{
		err: domainerrors.NotFound("not_found", "workspace not found"),
	}
	middleware := NewWorkspaceContextMiddleware(resolver)
	app := fiber.New()
	app.Get("/", middleware.RequireWorkspace(), func(*fiber.Ctx) error { return nil })

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(WorkspaceHeader, "  unknown  ")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusNotFound)
	}
	var body respond.ErrorResponse
	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "workspace_not_found" {
		t.Fatalf("code = %q, want workspace_not_found", body.Code)
	}
	if len(resolver.identities) != 1 || resolver.identities[0] != "unknown" {
		t.Fatalf("resolver identities = %#v, want [unknown]", resolver.identities)
	}
}

func TestRequireWorkspaceRejectsMissingHeader(t *testing.T) {
	resolver := &workspaceResolverStub{workspace: &entities.RWorkspace{ID: uuid.New()}}
	middleware := NewWorkspaceContextMiddleware(resolver)
	app := fiber.New()
	app.Get("/", middleware.RequireWorkspace(), func(*fiber.Ctx) error { return nil })
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
	if len(resolver.identities) != 0 {
		t.Fatalf("resolver identities = %#v, want no calls", resolver.identities)
	}
}

func TestRequireWorkspaceRejectsBlankHeader(t *testing.T) {
	resolver := &workspaceResolverStub{workspace: &entities.RWorkspace{ID: uuid.New()}}
	middleware := NewWorkspaceContextMiddleware(resolver)
	app := fiber.New()
	app.Get("/", middleware.RequireWorkspace(), func(*fiber.Ctx) error { return nil })

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(WorkspaceHeader, "   ")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
	var body respond.ErrorResponse
	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "workspace_required" {
		t.Fatalf("code = %q, want workspace_required", body.Code)
	}
	if len(resolver.identities) != 0 {
		t.Fatalf("resolver identities = %#v, want no calls", resolver.identities)
	}
}

func TestRequireWorkspaceAttachesResolvedID(t *testing.T) {
	wantID := uuid.New()
	middleware := NewWorkspaceContextMiddleware(&workspaceResolverStub{workspace: &entities.RWorkspace{ID: wantID, Identity: "default"}})
	app := fiber.New()
	app.Get("/", middleware.RequireWorkspace(), func(c *fiber.Ctx) error {
		scope, ok := entities.WorkspaceFromContext(c.UserContext())
		if !ok || scope.ID != wantID || scope.Identity != "default" {
			t.Fatalf("workspace scope = %#v, ok=%v", scope, ok)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(WorkspaceHeader, "default")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusNoContent)
	}
}
