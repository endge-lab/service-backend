package folder

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestFolderHandlers(t *testing.T) {
	service := &folderServiceStub{
		folder: &entities.RFolder{
			ID:          uuid.New(),
			EntityType:  entities.FolderEntityTypeComponentsLegacy,
			Identity:    "shared-components-legacy",
			DisplayName: "Shared legacy components",
			Meta:        map[string]any{},
		},
	}
	handler := &Handler{
		folderService: service,
		validator:     appvalidator.NewValidator(),
		logger:        zap.NewNop(),
	}
	app := fiber.New()
	folders := app.Group("/api/v1/projects/:project_identity/folders")
	folders.Post("/", handler.CreateFolder)
	folders.Get("/", handler.ListFolders)
	folders.Get("/:folder_identity", handler.GetFolderByIdentity)
	folders.Patch("/:folder_identity", handler.UpdateFolder)
	folders.Delete("/:folder_identity", handler.SoftDeleteFolder)
	folders.Post("/:folder_identity/restore", handler.RestoreFolder)

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
			path:       "/api/v1/projects/demo-project/folders/",
			body:       `{"entityType":"components-legacy","identity":"shared-components-legacy","displayName":"Shared legacy components","parentIdentity":"root-components-legacy","meta":{}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "list",
			method:     http.MethodGet,
			path:       "/api/v1/projects/demo-project/folders/?entity_type=components-legacy",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get",
			method:     http.MethodGet,
			path:       "/api/v1/projects/demo-project/folders/shared-components-legacy?entity_type=components-legacy",
			wantStatus: http.StatusOK,
		},
		{
			name:       "update",
			method:     http.MethodPatch,
			path:       "/api/v1/projects/demo-project/folders/shared-components-legacy?entity_type=components-legacy",
			body:       `{"displayName":"Updated legacy components","parentIdentity":"root-components-legacy","meta":{}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "soft delete",
			method:     http.MethodDelete,
			path:       "/api/v1/projects/demo-project/folders/shared-components-legacy?entity_type=components-legacy",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "restore",
			method:     http.MethodPost,
			path:       "/api/v1/projects/demo-project/folders/shared-components-legacy/restore?entity_type=components-legacy",
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

type folderServiceStub struct {
	folder *entities.RFolder
}

func (s *folderServiceStub) Create(context.Context, folders.CreateFolderInput) (*entities.RFolder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) Update(context.Context, folders.UpdateFolderInput) (*entities.RFolder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) GetByID(context.Context, uuid.UUID) (*entities.RFolder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) GetByIdentity(
	context.Context,
	folders.GetFolderInput,
) (*entities.RFolder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) List(
	context.Context,
	folders.ListFoldersInput,
) ([]*entities.RFolder, error) {
	return []*entities.RFolder{s.folder}, nil
}

func (s *folderServiceStub) SoftDelete(context.Context, folders.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) Restore(context.Context, folders.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) HardDelete(context.Context, folders.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) Count(context.Context, folders.ListFoldersInput) (int64, error) {
	return 1, nil
}
