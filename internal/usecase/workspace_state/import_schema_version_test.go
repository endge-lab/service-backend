package workspace_state

import (
	"errors"
	"testing"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// TestValidateWorkspaceSchemaVersion проверяет строгую совместимость полного workspace snapshot.
func TestValidateWorkspaceSchemaVersion(t *testing.T) {
	if err := validateWorkspaceSchemaVersion(2, 2); err != nil {
		t.Fatalf("expected matching schema version to pass: %v", err)
	}

	err := validateWorkspaceSchemaVersion(1, 2)
	if err == nil {
		t.Fatal("expected mismatching schema version to fail")
	}
	if code := domainerrors.CodeOf(err); code != "workspace_schema_unsupported" {
		t.Fatalf("unexpected error code %q", code)
	}
	if !errors.Is(err, domainerrors.ErrInvalidInput) {
		t.Fatal("expected invalid input error")
	}
	details := domainerrors.DetailsOf(err)
	if details["exportedSchemaVersion"] != 1 || details["supportedSchemaVersion"] != 2 {
		t.Fatalf("unexpected error details: %#v", details)
	}
}
