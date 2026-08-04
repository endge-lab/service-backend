package shared

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// FuzzDecodeAndValidate проверяет, что произвольный body не вызывает panic и не обходит transport validation.
func FuzzDecodeAndValidate(f *testing.F) {
	for _, seed := range []string{`{"identity":"query"}`, `{}`, `{"identity":"query","legacy":true}`, `{"identity":`, `null`, `[]`, `{"identity":"query"}{}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 64*1024 {
			t.Skip()
		}
		app := fiber.New()
		useCaseCalled := false
		app.Post("/", func(c *fiber.Ctx) error {
			_, err := DecodeAndValidate[validationRequest](c, appvalidator.NewValidator())
			if err != nil {
				return respond.WriteErrorResponse(c, err)
			}
			useCaseCalled = true
			return c.SendStatus(fiber.StatusNoContent)
		})
		request := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(body))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request, -1)
		if err != nil {
			t.Fatalf("transport вернул неконтролируемую ошибку: %v", err)
		}
		if response.StatusCode != fiber.StatusNoContent && response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("неожиданный status=%d", response.StatusCode)
		}
		if response.StatusCode == fiber.StatusBadRequest && useCaseCalled {
			t.Fatal("невалидный transport input дошёл до use case")
		}
	})
}

// FuzzIfMatch проверяет произвольные ETag, кавычки и числовые переполнения.
func FuzzIfMatch(f *testing.F) {
	for _, seed := range []string{"", `"1"`, "1", `W/"3"`, "-1", "0", "922337203685477580799", "not-an-etag"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, header string) {
		if len(header) > 4096 {
			t.Skip()
		}
		app := fiber.New()
		app.Get("/", func(c *fiber.Ctx) error {
			if _, err := IfMatch(c); err != nil {
				return respond.WriteErrorResponse(c, err)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})
		request := httptest.NewRequest(fiber.MethodGet, "/", nil)
		request.Header.Set(fiber.HeaderIfMatch, header)
		response, err := app.Test(request, -1)
		if err != nil {
			t.Fatalf("If-Match вызвал неконтролируемую ошибку: %v", err)
		}
		if response.StatusCode != fiber.StatusNoContent && response.StatusCode != fiber.StatusBadRequest && response.StatusCode != fiber.StatusPreconditionRequired {
			t.Fatalf("неожиданный status=%d для header=%q", response.StatusCode, header)
		}
	})
}
