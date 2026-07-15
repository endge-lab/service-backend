package converter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestConverterHandlers(t *testing.T) {
	value := &adapters.ConverterWithFolder{
		Converter:      &entities.Converter{ID: uuid.New(), Identity: "date-format", ConverterType: "format"},
		FolderIdentity: "root-converters",
	}
	service := &converterServiceStub{value: value}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	routes := app.Group("/api/v1/projects/:project_identity/converters")
	routes.Post("/", handler.Create)
	routes.Get("/", handler.List)
	routes.Get("/:converter_identity", handler.GetByIdentity)
	routes.Patch("/:converter_identity", handler.Update)
	routes.Delete("/:converter_identity", handler.SoftDelete)
	routes.Post("/:converter_identity/restore", handler.Restore)
	routes.Delete("/:converter_identity/hard", handler.HardDelete)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"create", http.MethodPost, "/api/v1/projects/demo/converters/", `{"folderIdentity":"root-converters","identity":"date-format","displayName":"Date format","converterType":"format","source":{}}`, http.StatusCreated},
		{"list", http.MethodGet, "/api/v1/projects/demo/converters/?folder_identity=root-converters", "", http.StatusOK},
		{"get", http.MethodGet, "/api/v1/projects/demo/converters/date-format", "", http.StatusOK},
		{"update", http.MethodPatch, "/api/v1/projects/demo/converters/date-format", `{"folderIdentity":"root-converters","displayName":"Date format","converterType":"format","source":{}}`, http.StatusOK},
		{"soft delete", http.MethodDelete, "/api/v1/projects/demo/converters/date-format", "", http.StatusNoContent},
		{"restore", http.MethodPost, "/api/v1/projects/demo/converters/date-format/restore", "", http.StatusNoContent},
		{"hard delete", http.MethodDelete, "/api/v1/projects/demo/converters/date-format/hard", "", http.StatusNoContent},
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
				if !strings.Contains(string(body), `"folderIdentity":"root-converters"`) {
					t.Fatalf("response does not contain folder identity: %s", body)
				}
			}
		})
	}

	if service.createInput.ProjectIdentity != "demo" || service.listInput.FolderIdentity == nil || *service.listInput.FolderIdentity != "root-converters" {
		t.Fatalf("unexpected mapped inputs: %#v %#v", service.createInput, service.listInput)
	}
}

func TestConverterHandlerValidationAndDomainErrors(t *testing.T) {
	service := &converterServiceStub{value: &adapters.ConverterWithFolder{Converter: &entities.Converter{ID: uuid.New()}, FolderIdentity: "root-converters"}}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	app.Post("/converters", handler.Create)
	app.Get("/converters/:converter_identity", handler.GetByIdentity)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		err        error
		wantStatus int
	}{
		{"invalid json", http.MethodPost, "/converters", `{`, nil, http.StatusBadRequest},
		{"validation", http.MethodPost, "/converters", `{}`, nil, http.StatusBadRequest},
		{"conflict", http.MethodPost, "/converters", `{"folderIdentity":"root-converters","identity":"date-format","displayName":"Date format","converterType":"format","source":{}}`, apperrors.Conflict("identity_conflict", "identity conflict"), http.StatusConflict},
		{"not found", http.MethodGet, "/converters/date-format", "", apperrors.NotFound("converter_not_found", "converter not found"), http.StatusNotFound},
		{"internal", http.MethodGet, "/converters/date-format", "", apperrors.Internal("internal_error", "internal error"), http.StatusInternalServerError},
	}

	for _, test := range tests {
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
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}

type converterServiceStub struct {
	value       *adapters.ConverterWithFolder
	err         error
	createInput adapters.CreateConverterInput
	listInput   adapters.ListConvertersInput
}

func (s *converterServiceStub) Create(_ context.Context, input adapters.CreateConverterInput) (*adapters.ConverterWithFolder, error) {
	s.createInput = input
	return s.value, s.err
}

func (s *converterServiceStub) Update(context.Context, adapters.UpdateConverterInput) (*adapters.ConverterWithFolder, error) {
	return s.value, s.err
}

func (s *converterServiceStub) GetByIdentity(context.Context, adapters.GetConverterInput) (*adapters.ConverterWithFolder, error) {
	return s.value, s.err
}

func (s *converterServiceStub) List(_ context.Context, input adapters.ListConvertersInput) ([]*adapters.ConverterWithFolder, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	return []*adapters.ConverterWithFolder{s.value}, nil
}

func (s *converterServiceStub) SoftDelete(context.Context, adapters.ConverterIdentityInput) error {
	return s.err
}

func (s *converterServiceStub) Restore(context.Context, adapters.ConverterIdentityInput) error {
	return s.err
}

func (s *converterServiceStub) HardDelete(context.Context, adapters.ConverterIdentityInput) error {
	return s.err
}

func (s *converterServiceStub) Count(context.Context, adapters.ListConvertersInput) (int64, error) {
	return 0, s.err
}
