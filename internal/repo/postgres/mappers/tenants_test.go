package mappers

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestTenantMapsConfigurationAndNullableFields(t *testing.T) {
	t.Parallel()

	folderID := uuid.New()
	description := "Tenant description"
	tenant, err := Tenant(sqlc.Tenant{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Identity:    "tenant-default",
		DisplayName: "Default tenant",
		Code:        "TENANT_DEFAULT",
		Description: pgtype.Text{String: description, Valid: true},
		FolderID:    pgtype.UUID{Bytes: folderID, Valid: true},
		Configuration: []byte(`{
			"mode":"inherit",
			"patch":{"defaultTheme":{"op":"set","value":"tenant-brand"}}
		}`),
	})
	if err != nil {
		t.Fatalf("map tenant: %v", err)
	}

	if tenant.Description == nil || *tenant.Description != description {
		t.Fatalf("description = %#v", tenant.Description)
	}
	if tenant.FolderID == nil || *tenant.FolderID != folderID {
		t.Fatalf("folder ID = %#v", tenant.FolderID)
	}
	if tenant.Configuration.Mode != entities.EndgeConfigurationContributionModeInherit {
		t.Fatalf("configuration mode = %q", tenant.Configuration.Mode)
	}
	if got := string(tenant.Configuration.Patch[entities.EndgeConfigurationPatchKeyDefaultTheme]); got != `{"op":"set","value":"tenant-brand"}` {
		t.Fatalf("default theme patch = %s", got)
	}
}

func TestTenantRejectsInvalidConfigurationJSON(t *testing.T) {
	t.Parallel()

	if _, err := Tenant(sqlc.Tenant{Configuration: []byte(`{"mode":`)}); err == nil {
		t.Fatal("expected invalid configuration JSON error")
	}
}
