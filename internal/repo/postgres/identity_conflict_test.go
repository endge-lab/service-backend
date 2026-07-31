package postgres

import (
	"testing"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapProjectIdentityConflict(t *testing.T) {
	err := mapProjectWriteError(&pgconn.PgError{
		Code:           postgresUniqueViolation,
		ConstraintName: "projects_identity_unique",
	})

	if code := domainerrors.CodeOf(err); code != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", code)
	}
}

func TestMapFolderIdentityConflict(t *testing.T) {
	err := mapFolderStorageError(&pgconn.PgError{
		Code:           postgresUniqueViolation,
		ConstraintName: "folders_project_entity_identity_unique",
	}, "internal_error")

	if code := domainerrors.CodeOf(err); code != "identity_conflict" {
		t.Fatalf("error code = %q, want identity_conflict", code)
	}
}

func TestMapTenantIdentityAndCodeConflicts(t *testing.T) {
	for constraint, wantCode := range map[string]string{
		"tenants_workspace_identity_unique": "tenant_identity_conflict",
		"tenants_workspace_code_unique":     "tenant_code_conflict",
	} {
		err := mapTenantWriteError(&pgconn.PgError{
			Code:           postgresUniqueViolation,
			ConstraintName: constraint,
		})

		if code := domainerrors.CodeOf(err); code != wantCode {
			t.Errorf("constraint %s: error code = %q, want %s", constraint, code, wantCode)
		}
	}
}

func TestMapTenantDeleteForeignKeyViolation(t *testing.T) {
	err := mapTenantDeleteError(&pgconn.PgError{Code: postgresForeignKeyViolation})
	if code := domainerrors.CodeOf(err); code != "tenant_in_use" {
		t.Fatalf("error code = %q, want tenant_in_use", code)
	}
}

func TestTenantNotFoundError(t *testing.T) {
	if code := domainerrors.CodeOf(tenantNotFoundError()); code != "tenant_not_found" {
		t.Fatalf("error code = %q, want tenant_not_found", code)
	}
}
