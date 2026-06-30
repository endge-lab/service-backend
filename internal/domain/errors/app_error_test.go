package errors

import (
	"errors"
	"testing"
)

func TestCodeOfReturnsSpecificWrappedCode(t *testing.T) {
	err := InvalidInput("todo.invalid_title", "Некорректный заголовок задачи")

	if got := CodeOf(err); got != "todo.invalid_title" {
		t.Fatalf("CodeOf() = %q, want %q", got, "todo.invalid_title")
	}
	if got := HTTPStatusOf(err); got != 400 {
		t.Fatalf("HTTPStatusOf() = %d, want 400", got)
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatal("expected wrapped error to match ErrInvalidInput")
	}
}

func TestWithDetailsPreservesAppError(t *testing.T) {
	err := WithDetails(ErrAuthUserIDRequired, map[string]any{"field": "authUserId"})

	if got := CodeOf(err); got != "session.auth_user_id_required" {
		t.Fatalf("CodeOf() = %q, want %q", got, "session.auth_user_id_required")
	}
	if got := DetailsOf(err)["field"]; got != "authUserId" {
		t.Fatalf("DetailsOf()[field] = %v, want authUserId", got)
	}
}
