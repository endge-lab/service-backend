package access_control

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/gofiber/fiber/v2"
)

type accessUseCaseStub struct{}

func (accessUseCaseStub) SearchUsers(context.Context, string, string, string, int) (resourceusecase.Page[entities.AccessGrantUser], error) {
	return resourceusecase.Page[entities.AccessGrantUser]{Items: []entities.AccessGrantUser{}}, nil
}
func (accessUseCaseStub) List(context.Context, resourceusecase.ListInput) (resourceusecase.Page[entities.AccessGrant], error) {
	return resourceusecase.Page[entities.AccessGrant]{Items: []entities.AccessGrant{}}, nil
}
func (accessUseCaseStub) Put(context.Context, resourceusecase.PutInput) (*entities.AccessGrant, error) {
	return &entities.AccessGrant{}, nil
}
func (accessUseCaseStub) Delete(context.Context, string) error { return nil }
func (accessUseCaseStub) Bulk(context.Context, resourceusecase.BulkInput) (resourceusecase.BulkResult, error) {
	return resourceusecase.BulkResult{}, nil
}

func TestRoutesDoNotRequireWorkspaceHeader(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/api/v1"), NewHandler(accessUseCaseStub{}, nil))

	for _, path := range []string{
		"/api/v1/service-users/search?q=iv",
		"/api/v1/access-grants?scopeType=platform",
	} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, response.StatusCode)
		}
	}
}
