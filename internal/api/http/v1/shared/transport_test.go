package shared

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type validationRequest struct {
	Identity string `json:"identity" validate:"required,max=10"`
}

// TestDecodeAndValidateContract проверяет JSON decoder, неизвестные поля и единый validation envelope.
func TestDecodeAndValidateContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantField  string
	}{
		{name: "valid", body: `{"identity":"query"}`, wantStatus: fiber.StatusNoContent},
		{name: "unknown field", body: `{"identity":"query","legacy":true}`, wantStatus: fiber.StatusBadRequest},
		{name: "missing required", body: `{}`, wantStatus: fiber.StatusBadRequest, wantField: "identity"},
		{name: "trailing json", body: `{"identity":"query"}{}`, wantStatus: fiber.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Post("/", func(c *fiber.Ctx) error {
				_, err := DecodeAndValidate[validationRequest](c, appvalidator.NewValidator())
				if err != nil {
					return respond.WriteErrorResponse(c, err)
				}
				return c.SendStatus(fiber.StatusNoContent)
			})
			request := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if tt.wantField != "" {
				var body struct {
					Code    string `json:"code"`
					Details struct {
						Fields map[string]string `json:"fields"`
					} `json:"details"`
				}
				if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if body.Code != "validation_error" || body.Details.Fields[tt.wantField] == "" {
					t.Fatalf("validation response = %#v", body)
				}
			}
		})
	}
}

// TestParseDocumentFilterRejectsUnsafePagination проверяет границы limit и offset.
func TestParseDocumentFilterRejectsUnsafePagination(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		_, err := ParseDocumentFilter(c)
		if err != nil {
			return respond.WriteErrorResponse(c, err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	for _, target := range []string{"/?limit=0", "/?limit=501", "/?offset=-1"} {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, target, nil))
		if err != nil {
			t.Fatalf("request %s: %v", target, err)
		}
		if response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", target, response.StatusCode)
		}
	}
}
