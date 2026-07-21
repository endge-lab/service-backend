package component_legacy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/components_legacy"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestComponentLegacyHandlers(t *testing.T) {
	value := &components_legacy.ComponentLegacyWithFolder{
		ComponentLegacy: &entities.RComponentLegacy{ID: uuid.New(), Identity: "user-card", ComponentType: entities.RComponentLegacyTypeSFC, Source: "<template />"},
		FolderIdentity:  "root-components-legacy",
	}
	service := &componentServiceStub{value: value}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	routes := app.Group("/api/v1/projects/:project_identity/components-legacy")
	routes.Post("/", handler.Create)
	routes.Get("/", handler.List)
	routes.Get("/:component_identity", handler.GetByIdentity)
	routes.Patch("/:component_identity", handler.Update)
	routes.Delete("/:component_identity", handler.SoftDelete)
	routes.Post("/:component_identity/restore", handler.Restore)
	routes.Delete("/:component_identity/hard", handler.HardDelete)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"create", http.MethodPost, "/api/v1/projects/demo/components-legacy/", `{"folderIdentity":"root-components-legacy","identity":"user-card","displayName":"User card","componentType":"component-sfc","source":"<template />"}`, http.StatusCreated},
		{"list", http.MethodGet, "/api/v1/projects/demo/components-legacy/?folder_identity=root-components-legacy&component_type=component-sfc", "", http.StatusOK},
		{"get", http.MethodGet, "/api/v1/projects/demo/components-legacy/user-card", "", http.StatusOK},
		{"update", http.MethodPatch, "/api/v1/projects/demo/components-legacy/user-card", `{"folderIdentity":"root-components-legacy","displayName":"User card","componentType":"component-sfc","source":"<template />"}`, http.StatusOK},
		{"soft delete", http.MethodDelete, "/api/v1/projects/demo/components-legacy/user-card", "", http.StatusNoContent},
		{"restore", http.MethodPost, "/api/v1/projects/demo/components-legacy/user-card/restore", "", http.StatusNoContent},
		{"hard delete", http.MethodDelete, "/api/v1/projects/demo/components-legacy/user-card/hard", "", http.StatusNoContent},
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
				if !strings.Contains(string(body), `"folderIdentity":"root-components-legacy"`) {
					t.Fatalf("response does not contain folder identity: %s", body)
				}
			}
		})
	}

	if service.createInput.ProjectIdentity != "demo" || service.listInput.FolderIdentity == nil || *service.listInput.FolderIdentity != "root-components-legacy" || service.listInput.ComponentType == nil || *service.listInput.ComponentType != entities.RComponentLegacyTypeSFC {
		t.Fatalf("unexpected mapped inputs: %#v %#v", service.createInput, service.listInput)
	}
}

func TestComponentLegacyHandlerValidationAndDomainErrors(t *testing.T) {
	service := &componentServiceStub{value: &components_legacy.ComponentLegacyWithFolder{ComponentLegacy: &entities.RComponentLegacy{ID: uuid.New()}, FolderIdentity: "root-components-legacy"}}
	handler := &Handler{service: service, validator: appvalidator.NewValidator(), logger: zap.NewNop()}
	app := fiber.New()
	app.Post("/components-legacy", handler.Create)
	app.Get("/components-legacy/:component_identity", handler.GetByIdentity)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		err        error
		wantStatus int
	}{
		{"invalid json", http.MethodPost, "/components-legacy", `{`, nil, http.StatusBadRequest},
		{"validation", http.MethodPost, "/components-legacy", `{}`, nil, http.StatusBadRequest},
		{"unsupported component type", http.MethodPost, "/components-legacy", `{"folderIdentity":"root-components-legacy","identity":"user-card","displayName":"User card","componentType":"legacy-html","source":"<template />"}`, nil, http.StatusBadRequest},
		{"not found", http.MethodGet, "/components-legacy/user-card", "", apperrors.NotFound("component_not_found", "component not found"), http.StatusNotFound},
		{"internal", http.MethodGet, "/components-legacy/user-card", "", apperrors.Internal("internal_error", "internal error"), http.StatusInternalServerError},
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

type componentServiceStub struct {
	value       *components_legacy.ComponentLegacyWithFolder
	err         error
	createInput components_legacy.CreateComponentLegacyInput
	listInput   components_legacy.ListComponentsLegacyInput
}

func (s *componentServiceStub) Create(_ context.Context, input components_legacy.CreateComponentLegacyInput) (*components_legacy.ComponentLegacyWithFolder, error) {
	s.createInput = input
	return s.value, s.err
}

func (s *componentServiceStub) Update(context.Context, components_legacy.UpdateComponentLegacyInput) (*components_legacy.ComponentLegacyWithFolder, error) {
	return s.value, s.err
}

func (s *componentServiceStub) GetByIdentity(context.Context, components_legacy.GetComponentLegacyInput) (*components_legacy.ComponentLegacyWithFolder, error) {
	return s.value, s.err
}

func (s *componentServiceStub) List(_ context.Context, input components_legacy.ListComponentsLegacyInput) ([]*components_legacy.ComponentLegacyWithFolder, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	return []*components_legacy.ComponentLegacyWithFolder{s.value}, nil
}

func (s *componentServiceStub) SoftDelete(context.Context, components_legacy.ComponentLegacyIdentityInput) error {
	return s.err
}

func (s *componentServiceStub) Restore(context.Context, components_legacy.ComponentLegacyIdentityInput) error {
	return s.err
}

func (s *componentServiceStub) HardDelete(context.Context, components_legacy.ComponentLegacyIdentityInput) error {
	return s.err
}

func (s *componentServiceStub) Count(context.Context, components_legacy.ListComponentsLegacyInput) (int64, error) {
	return 0, s.err
}
