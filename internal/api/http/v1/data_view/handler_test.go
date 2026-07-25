package data_view

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestDataViewHandlers(t *testing.T) {
	value := &data_views.DataViewWithRelations{DataView: &entities.RDataView{ID: uuid.New(), Identity: "users-table", ViewType: "pipeline", Source: map[string]any{}}, FolderIdentity: "root-data-views", QueryIdentity: "users-list"}
	service := &dataViewServiceStub{value: value}
	handler := &Handler{service: service, validator: appvalidator.NewValidator()}
	app := fiber.New()
	routes := app.Group("/api/v1/projects/:project_identity/data-views")
	routes.Post("/", handler.Create)
	routes.Get("/", handler.List)
	routes.Get("/:data_view_identity", handler.GetByIdentity)
	routes.Patch("/:data_view_identity", handler.Update)
	routes.Delete("/:data_view_identity", handler.SoftDelete)
	routes.Post("/:data_view_identity/restore", handler.Restore)
	routes.Delete("/:data_view_identity/hard", handler.HardDelete)

	tests := []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{"create", http.MethodPost, "/api/v1/projects/demo/data-views/", `{"folderIdentity":"root-data-views","queryIdentity":"users-list","identity":"users-table","displayName":"Users","viewType":"pipeline","source":{}}`, http.StatusCreated},
		{"list", http.MethodGet, "/api/v1/projects/demo/data-views/?folder_identity=root-data-views&query_identity=users-list", "", http.StatusOK},
		{"get", http.MethodGet, "/api/v1/projects/demo/data-views/users-table", "", http.StatusOK},
		{"update", http.MethodPatch, "/api/v1/projects/demo/data-views/users-table", `{"folderIdentity":"root-data-views","queryIdentity":"users-list","displayName":"Users","viewType":"pipeline","source":{}}`, http.StatusOK},
		{"soft delete", http.MethodDelete, "/api/v1/projects/demo/data-views/users-table", "", http.StatusNoContent},
		{"restore", http.MethodPost, "/api/v1/projects/demo/data-views/users-table/restore", "", http.StatusNoContent},
		{"hard delete", http.MethodDelete, "/api/v1/projects/demo/data-views/users-table/hard", "", http.StatusNoContent},
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
				if !strings.Contains(string(body), `"queryIdentity":"users-list"`) {
					t.Fatalf("response = %s", body)
				}
			}
		})
	}
	if service.createInput.ProjectIdentity != "demo" || service.listInput.FolderIdentity == nil || *service.listInput.FolderIdentity != "root-data-views" || service.listInput.QueryIdentity == nil || *service.listInput.QueryIdentity != "users-list" {
		t.Fatalf("unexpected inputs: %#v %#v", service.createInput, service.listInput)
	}
}

func TestDataViewHandlerValidationAndDomainErrors(t *testing.T) {
	service := &dataViewServiceStub{value: &data_views.DataViewWithRelations{DataView: &entities.RDataView{}, FolderIdentity: "root-data-views", QueryIdentity: "users-list"}}
	handler := &Handler{service: service, validator: appvalidator.NewValidator()}
	app := fiber.New()
	app.Post("/data-views", handler.Create)
	app.Get("/data-views/:data_view_identity", handler.GetByIdentity)
	for _, test := range []struct {
		name, method, path, body string
		err                      error
		want                     int
	}{
		{"invalid json", http.MethodPost, "/data-views", `{`, nil, http.StatusBadRequest},
		{"validation", http.MethodPost, "/data-views", `{}`, nil, http.StatusBadRequest},
		{"not found", http.MethodGet, "/data-views/users-table", "", apperrors.NotFound("not_found", "data view not found"), http.StatusNotFound},
		{"internal", http.MethodGet, "/data-views/users-table", "", apperrors.Internal("internal_error", "failure"), http.StatusInternalServerError},
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

type dataViewServiceStub struct {
	value       *data_views.DataViewWithRelations
	err         error
	createInput data_views.CreateDataViewInput
	listInput   data_views.ListDataViewsInput
}

func (s *dataViewServiceStub) Create(_ context.Context, input data_views.CreateDataViewInput) (*data_views.DataViewWithRelations, error) {
	s.createInput = input
	return s.value, s.err
}
func (s *dataViewServiceStub) Update(context.Context, data_views.UpdateDataViewInput) (*data_views.DataViewWithRelations, error) {
	return s.value, s.err
}
func (s *dataViewServiceStub) GetByIdentity(context.Context, data_views.GetDataViewInput) (*data_views.DataViewWithRelations, error) {
	return s.value, s.err
}
func (s *dataViewServiceStub) List(_ context.Context, input data_views.ListDataViewsInput) ([]*data_views.DataViewWithRelations, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	return []*data_views.DataViewWithRelations{s.value}, nil
}
func (s *dataViewServiceStub) SoftDelete(context.Context, data_views.DataViewIdentityInput) error {
	return s.err
}
func (s *dataViewServiceStub) Restore(context.Context, data_views.DataViewIdentityInput) error {
	return s.err
}
func (s *dataViewServiceStub) HardDelete(context.Context, data_views.DataViewIdentityInput) error {
	return s.err
}
func (s *dataViewServiceStub) Count(context.Context, data_views.ListDataViewsInput) (int64, error) {
	return 0, s.err
}
