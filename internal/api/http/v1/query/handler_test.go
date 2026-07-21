package query

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestQueryHandlers(t *testing.T) {
	value := &queries.QueryWithFolder{Query: &entities.RQuery{ID: uuid.New(), Identity: "users-list", QueryType: "http", Source: map[string]any{}}, FolderIdentity: "root-queries"}
	service := &queryServiceStub{value: value}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	routes := app.Group("/api/v1/projects/:project_identity/queries")
	routes.Post("/", handler.Create)
	routes.Get("/", handler.List)
	routes.Get("/:query_identity", handler.GetByIdentity)
	routes.Patch("/:query_identity", handler.Update)
	routes.Delete("/:query_identity", handler.SoftDelete)
	routes.Post("/:query_identity/restore", handler.Restore)
	routes.Delete("/:query_identity/hard", handler.HardDelete)

	tests := []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{"create", http.MethodPost, "/api/v1/projects/demo/queries/", `{"folderIdentity":"root-queries","identity":"users-list","displayName":"Users","queryType":"http","source":{}}`, http.StatusCreated},
		{"list", http.MethodGet, "/api/v1/projects/demo/queries/?folder_identity=root-queries&query_type=http", "", http.StatusOK},
		{"get", http.MethodGet, "/api/v1/projects/demo/queries/users-list", "", http.StatusOK},
		{"update", http.MethodPatch, "/api/v1/projects/demo/queries/users-list", `{"folderIdentity":"root-queries","displayName":"Users","queryType":"http","source":{}}`, http.StatusOK},
		{"soft delete", http.MethodDelete, "/api/v1/projects/demo/queries/users-list", "", http.StatusNoContent},
		{"restore", http.MethodPost, "/api/v1/projects/demo/queries/users-list/restore", "", http.StatusNoContent},
		{"hard delete", http.MethodDelete, "/api/v1/projects/demo/queries/users-list/hard", "", http.StatusNoContent},
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
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			if test.wantStatus == http.StatusOK || test.wantStatus == http.StatusCreated {
				body, _ := io.ReadAll(response.Body)
				if !strings.Contains(string(body), `"folderIdentity":"root-queries"`) {
					t.Fatalf("response = %s", body)
				}
			}
		})
	}
	if service.createInput.ProjectIdentity != "demo" || service.listInput.FolderIdentity == nil || *service.listInput.FolderIdentity != "root-queries" || service.listInput.QueryType == nil || *service.listInput.QueryType != "http" {
		t.Fatalf("unexpected inputs: %#v %#v", service.createInput, service.listInput)
	}
}

func TestQueryHandlerValidationAndDomainErrors(t *testing.T) {
	service := &queryServiceStub{value: &queries.QueryWithFolder{Query: &entities.RQuery{}, FolderIdentity: "root-queries"}}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	app.Post("/queries", handler.Create)
	app.Get("/queries/:query_identity", handler.GetByIdentity)
	for _, test := range []struct {
		name, method, path, body string
		err                      error
		want                     int
	}{
		{"invalid json", http.MethodPost, "/queries", `{`, nil, http.StatusBadRequest},
		{"validation", http.MethodPost, "/queries", `{}`, nil, http.StatusBadRequest},
		{"timeout must be positive", http.MethodPost, "/queries", `{"folderIdentity":"root-queries","identity":"users-list","displayName":"Users","queryType":"http","source":{},"timeoutMs":0}`, nil, http.StatusBadRequest},
		{"not found", http.MethodGet, "/queries/users-list", "", apperrors.NotFound("not_found", "query not found"), http.StatusNotFound},
		{"internal", http.MethodGet, "/queries/users-list", "", apperrors.Internal("internal_error", "failure"), http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			service.err = test.err
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
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
		})
	}
}

type queryServiceStub struct {
	value       *queries.QueryWithFolder
	err         error
	createInput queries.CreateQueryInput
	listInput   queries.ListQueriesInput
}

func (s *queryServiceStub) Create(_ context.Context, input queries.CreateQueryInput) (*queries.QueryWithFolder, error) {
	s.createInput = input
	return s.value, s.err
}
func (s *queryServiceStub) Update(context.Context, queries.UpdateQueryInput) (*queries.QueryWithFolder, error) {
	return s.value, s.err
}
func (s *queryServiceStub) GetByIdentity(context.Context, queries.GetQueryInput) (*queries.QueryWithFolder, error) {
	return s.value, s.err
}
func (s *queryServiceStub) List(_ context.Context, input queries.ListQueriesInput) ([]*queries.QueryWithFolder, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	return []*queries.QueryWithFolder{s.value}, nil
}
func (s *queryServiceStub) SoftDelete(context.Context, queries.QueryIdentityInput) error {
	return s.err
}
func (s *queryServiceStub) Restore(context.Context, queries.QueryIdentityInput) error { return s.err }
func (s *queryServiceStub) HardDelete(context.Context, queries.QueryIdentityInput) error {
	return s.err
}
func (s *queryServiceStub) Count(context.Context, queries.ListQueriesInput) (int64, error) {
	return 0, s.err
}
