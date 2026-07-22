package project

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestProjectHandlers(t *testing.T) {
	service := &projectServiceStub{
		project: &entities.RProject{
			ID:          uuid.New(),
			Identity:    "demo-project",
			DisplayName: "Demo Project",
			Active:      true,
			Meta:        map[string]any{},
		},
	}
	handler := &Handler{
		projectService: service,
		validator:      appvalidator.NewValidator(),
	}
	app := fiber.New()
	projects := app.Group("/api/v1/projects")
	projects.Post("/", handler.CreateProject)
	projects.Get("/", handler.ListProjects)
	projects.Get("/:project_identity", handler.GetProjectByIdentity)
	projects.Patch("/:project_identity", handler.UpdateProject)
	projects.Delete("/:project_identity", handler.SoftDeleteProject)
	projects.Post("/:project_identity/restore", handler.RestoreProject)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "create",
			method:     http.MethodPost,
			path:       "/api/v1/projects/",
			body:       `{"identity":"demo-project","displayName":"Demo Project","active":true,"meta":{}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "list",
			method:     http.MethodGet,
			path:       "/api/v1/projects/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get",
			method:     http.MethodGet,
			path:       "/api/v1/projects/demo-project",
			wantStatus: http.StatusOK,
		},
		{
			name:       "update",
			method:     http.MethodPatch,
			path:       "/api/v1/projects/demo-project",
			body:       `{"displayName":"Updated Project","active":true,"meta":{}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "soft delete",
			method:     http.MethodDelete,
			path:       "/api/v1/projects/demo-project",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "restore",
			method:     http.MethodPost,
			path:       "/api/v1/projects/demo-project/restore",
			wantStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			request.Host = "service-backend.test"
			if test.body != "" {
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			}

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}

type projectServiceStub struct {
	project *entities.RProject
}

func (s *projectServiceStub) Create(context.Context, projects.CreateProjectInput) (*entities.RProject, error) {
	return s.project, nil
}

func (s *projectServiceStub) Update(context.Context, projects.UpdateProjectInput) (*entities.RProject, error) {
	return s.project, nil
}

func (s *projectServiceStub) GetByID(context.Context, uuid.UUID) (*entities.RProject, error) {
	return s.project, nil
}

func (s *projectServiceStub) GetByIdentity(context.Context, string) (*entities.RProject, error) {
	return s.project, nil
}

func (s *projectServiceStub) List(context.Context) ([]*entities.RProject, error) {
	return []*entities.RProject{s.project}, nil
}

func (s *projectServiceStub) SoftDelete(context.Context, string) error {
	return nil
}

func (s *projectServiceStub) Restore(context.Context, string) error {
	return nil
}

func (s *projectServiceStub) HardDelete(context.Context, string) error {
	return nil
}

func (s *projectServiceStub) Count(context.Context) (int64, error) {
	return 1, nil
}
