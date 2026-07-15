package folder

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestFolderHandlers(t *testing.T) {
	service := &folderServiceStub{
		folder: &entities.Folder{
			ID:          uuid.New(),
			EntityType:  entities.FolderEntityTypeComponents,
			Identity:    "shared-components",
			DisplayName: "Shared Components",
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
			body:       `{"entityType":"components","identity":"shared-components","displayName":"Shared Components","parentIdentity":"root-components","meta":{}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "list",
			method:     http.MethodGet,
			path:       "/api/v1/projects/demo-project/folders/?entity_type=components",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get",
			method:     http.MethodGet,
			path:       "/api/v1/projects/demo-project/folders/shared-components?entity_type=components",
			wantStatus: http.StatusOK,
		},
		{
			name:       "update",
			method:     http.MethodPatch,
			path:       "/api/v1/projects/demo-project/folders/shared-components?entity_type=components",
			body:       `{"displayName":"Updated Components","parentIdentity":"root-components","meta":{}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "soft delete",
			method:     http.MethodDelete,
			path:       "/api/v1/projects/demo-project/folders/shared-components?entity_type=components",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "restore",
			method:     http.MethodPost,
			path:       "/api/v1/projects/demo-project/folders/shared-components/restore?entity_type=components",
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
	folder *entities.Folder
}

func (s *folderServiceStub) Create(context.Context, adapters.CreateFolderInput) (*entities.Folder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) Update(context.Context, adapters.UpdateFolderInput) (*entities.Folder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) GetByID(context.Context, uuid.UUID) (*entities.Folder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) GetByIdentity(
	context.Context,
	adapters.GetFolderInput,
) (*entities.Folder, error) {
	return s.folder, nil
}

func (s *folderServiceStub) List(
	context.Context,
	adapters.ListFoldersInput,
) ([]*entities.Folder, error) {
	return []*entities.Folder{s.folder}, nil
}

func (s *folderServiceStub) SoftDelete(context.Context, adapters.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) Restore(context.Context, adapters.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) HardDelete(context.Context, adapters.FolderIdentityInput) error {
	return nil
}

func (s *folderServiceStub) Count(context.Context, adapters.ListFoldersInput) (int64, error) {
	return 1, nil
}
