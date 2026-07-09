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
