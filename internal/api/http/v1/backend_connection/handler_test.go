package backend_connection

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/backend_connections"
	"github.com/gofiber/fiber/v2"
)

type listUseCaseStub struct{}

func (listUseCaseStub) List(context.Context) (resourceusecase.ListResult, error) {
	return resourceusecase.ListResult{Items: []entities.BackendConnection{}, CanManage: false}, nil
}
func (listUseCaseStub) Create(context.Context, string, string) (*entities.BackendConnection, error) {
	return nil, nil
}
func (listUseCaseStub) Delete(context.Context, string) error { return nil }

func TestListRouteDoesNotRequireWorkspaceHeader(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/api/v1"), NewHandler(listUseCaseStub{}, nil))

	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/backend-connections", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
