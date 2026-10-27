package revision

import (
	"net/http/httptest"
	"testing"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

func TestParseListOptions(t *testing.T) {
	cases := map[string]struct {
		want ports.RevisionListOptions
		code string
	}{
		"":                                   {want: ports.RevisionListOptions{IncludeSnapshot: true}},
		"?snapshot=false&limit=20&offset=40": {want: ports.RevisionListOptions{Limit: 20, Offset: 40}},
		"?limit=0":                           {code: "limit_invalid"},
		"?limit=501":                         {code: "limit_invalid"},
		"?offset=-1":                         {code: "offset_invalid"},
		"?snapshot=maybe":                    {code: "snapshot_invalid"},
	}
	for query, tc := range cases {
		app := fiber.New()
		var got ports.RevisionListOptions
		var gotErr error
		app.Get("/", func(c *fiber.Ctx) error {
			got, gotErr = parseListOptions(c)
			return nil
		})
		if _, err := app.Test(httptest.NewRequest("GET", "/"+query, nil)); err != nil {
			t.Fatal(err)
		}
		if tc.code != "" {
			if domainerrors.CodeOf(gotErr) != tc.code {
				t.Fatalf("%q: expected %s, got %v", query, tc.code, gotErr)
			}
			continue
		}
		if gotErr != nil || got != tc.want {
			t.Fatalf("%q: got %+v %v, want %+v", query, got, gotErr, tc.want)
		}
	}
}
