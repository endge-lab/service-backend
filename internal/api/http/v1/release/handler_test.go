package release

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/releases"
	"github.com/gofiber/fiber/v2"
)

type releaseUseCaseStub struct {
	metadataCalls int
	metadataInput string
	metadata      entities.Release
	artifact      entities.ReleaseArtifact
	artifactCalls int
}

func (s *releaseUseCaseStub) Create(context.Context, resourceusecase.CreateInput) (*entities.Release, error) {
	return &s.metadata, nil
}
func (s *releaseUseCaseStub) List(context.Context) ([]entities.Release, error) {
	return []entities.Release{s.metadata}, nil
}
func (s *releaseUseCaseStub) Get(_ context.Context, identity string) (*entities.Release, error) {
	s.metadataCalls++
	s.metadataInput = identity
	return &s.metadata, nil
}
func (s *releaseUseCaseStub) GetArtifact(context.Context, entities.Release) (*entities.ReleaseArtifact, error) {
	s.artifactCalls++
	return &s.artifact, nil
}

func TestHandlerExportRejectsInvalidDownloadBeforeReadingRelease(t *testing.T) {
	stub := &releaseUseCaseStub{}
	app := fiber.New()
	app.Get("/releases/:identity/export", NewHandler(stub, nil).Export)

	request := httptest.NewRequest(fiber.MethodGet, "/releases/production/export?download=not-a-bool", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
	if stub.metadataCalls != 0 || stub.artifactCalls != 0 {
		t.Fatalf("invalid query read release: metadata=%d artifact=%d", stub.metadataCalls, stub.artifactCalls)
	}
}

func TestHandlerExportDecodesReleaseIdentity(t *testing.T) {
	stub := &releaseUseCaseStub{
		metadata: entities.Release{ID: "release-id", Identity: "Keycloak Auth Test", Checksum: "checksum"},
		artifact: entities.ReleaseArtifact{ReleaseID: "release-id", Identity: "Keycloak Auth Test", Checksum: "checksum", Data: []byte(`{}`)},
	}
	app := fiber.New()
	app.Get("/releases/:identity/export", NewHandler(stub, nil).Export)

	request := httptest.NewRequest(fiber.MethodGet, "/releases/Keycloak%20Auth%20Test/export", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if stub.metadataInput != "Keycloak Auth Test" {
		t.Fatalf("identity = %q, want %q", stub.metadataInput, "Keycloak Auth Test")
	}
}

func (s *releaseUseCaseStub) PlanRestore(context.Context, string) (*entities.ImportPlan, error) {
	return nil, nil
}
func (s *releaseUseCaseStub) Restore(context.Context, string, int64) (*entities.Commit, error) {
	return nil, nil
}

func TestHandlerExportConditionalGET(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ifNoneMatch   string
		wantStatus    int
		wantBody      string
		artifactCalls int
	}{
		{name: "matching strong ETag", ifNoneMatch: `"checksum"`, wantStatus: fiber.StatusNotModified},
		{name: "matching weak ETag", ifNoneMatch: `W/"checksum"`, wantStatus: fiber.StatusNotModified},
		{name: "matching ETag in list", ifNoneMatch: `"other", "checksum"`, wantStatus: fiber.StatusNotModified},
		{name: "wildcard ETag", ifNoneMatch: "*", wantStatus: fiber.StatusNotModified},
		{name: "stale ETag", ifNoneMatch: `"old"`, wantStatus: fiber.StatusOK, wantBody: `{"kind":"workspace-snapshot"}`, artifactCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stub := &releaseUseCaseStub{
				metadata: entities.Release{ID: "release-id", Identity: "production", Checksum: "checksum"},
				artifact: entities.ReleaseArtifact{ReleaseID: "release-id", Identity: "production", Checksum: "checksum", Data: []byte(`{"kind":"workspace-snapshot"}`)},
			}
			app := fiber.New()
			handler := NewHandler(stub, nil)
			app.Get("/releases/:identity/export", handler.Export)
			request := httptest.NewRequest(fiber.MethodGet, "/releases/production/export", nil)
			if tt.ifNoneMatch != "" {
				request.Header.Set(fiber.HeaderIfNoneMatch, tt.ifNoneMatch)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if response.Header.Get(fiber.HeaderETag) != `"checksum"` {
				t.Fatalf("ETag = %q", response.Header.Get(fiber.HeaderETag))
			}
			if response.Header.Get(fiber.HeaderCacheControl) != "private, no-cache" {
				t.Fatalf("Cache-Control = %q", response.Header.Get(fiber.HeaderCacheControl))
			}
			if response.Header.Get(fiber.HeaderVary) != "X-Endge-Workspace, Authorization, Cookie" {
				t.Fatalf("Vary = %q", response.Header.Get(fiber.HeaderVary))
			}
			if stub.artifactCalls != tt.artifactCalls {
				t.Fatalf("artifact calls = %d, want %d", stub.artifactCalls, tt.artifactCalls)
			}
			if string(body) != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
