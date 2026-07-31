package domain

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/dependencies"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestListUsagesMapsQueryAndResponse(t *testing.T) {
	service := &useCaseStub{result: entities.DomainDependencyUsages{
		Items:  []entities.DomainDependencyUsage{{OwnerType: "type", OwnerID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), OwnerIdentity: "OrderList", SourcePath: "schema.fields[0].type", VerificationState: entities.DomainDependencyVerificationStateVerified}},
		Total:  1,
		Limit:  25,
		Offset: 3,
	}}
	app := fiber.New()
	app.Get("/usages", newHandler(service).ListUsages)

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/usages?dependency_type=type&dependency_identity=Orders&limit=25&offset=3", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"ownerIdentity":"OrderList"`) || !strings.Contains(string(body), `"total":1`) {
		t.Fatalf("response body = %s", body)
	}
	if service.input.DependencyType != "type" || service.input.DependencyIdentity != "Orders" || service.input.Limit == nil || *service.input.Limit != 25 || service.input.Offset == nil || *service.input.Offset != 3 {
		t.Fatalf("input = %#v", service.input)
	}
}

func TestListUsagesReturnsValidationError(t *testing.T) {
	service := &useCaseStub{err: apperrors.InvalidInput("validation_error", "dependency type is required")}
	app := fiber.New()
	app.Get("/usages", newHandler(service).ListUsages)

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/usages?dependency_identity=Orders", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}

	response, err = app.Test(httptest.NewRequest(http.MethodGet, "/usages?dependency_type=type&dependency_identity=Orders&limit=bad", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed limit status = %d", response.StatusCode)
	}
}

func newHandler(service UseCase) *Handler {
	return NewHandler(service, observability.NewCore(otel.Tracer("domain-handler-test"), zap.NewNop()), nil)
}

type useCaseStub struct {
	input  dependencies.ListUsagesInput
	result entities.DomainDependencyUsages
	err    error
}

func (s *useCaseStub) ListUsages(_ context.Context, input dependencies.ListUsagesInput) (entities.DomainDependencyUsages, error) {
	s.input = input
	return s.result, s.err
}
