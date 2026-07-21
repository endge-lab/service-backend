package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"testing"
)

func TestWorkspaceJSONBRoundTrip(t *testing.T) {
	original := entities.DefaultEndgeConfiguration()
	params, err := CreateWorkspaceParams(&entities.RWorkspace{Identity: "default", DisplayName: "Default", Configuration: original})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Workspace(sqlc.Workspace{Identity: "default", DisplayName: "Default", Configuration: params.Configuration})
	if err != nil {
		t.Fatal(err)
	}
	if got.Configuration.DefaultLocale != "ru" || len(got.Configuration.Locales) != 2 {
		t.Fatalf("unexpected configuration: %+v", got.Configuration)
	}
}
func TestWorkspaceRejectsMalformedJSONB(t *testing.T) {
	if _, err := Workspace(sqlc.Workspace{Configuration: []byte("{")}); err == nil {
		t.Fatal("expected JSON error")
	}
}
