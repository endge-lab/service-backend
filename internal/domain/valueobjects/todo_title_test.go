package valueobjects

import (
	"errors"
	"strings"
	"testing"

	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
)

func TestNewTodoTitle(t *testing.T) {
	t.Run("normalizes valid title", func(t *testing.T) {
		title, err := NewTodoTitle("  Buy milk  ")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if title.Value() != "Buy milk" {
			t.Fatalf("expected normalized title, got %q", title.Value())
		}
	})

	t.Run("rejects empty title", func(t *testing.T) {
		_, err := NewTodoTitle("   ")
		if !errors.Is(err, domainerrors.ErrInvalidTodoTitle) {
			t.Fatalf("expected ErrInvalidTodoTitle, got %v", err)
		}
	})

	t.Run("rejects too long title", func(t *testing.T) {
		_, err := NewTodoTitle(strings.Repeat("a", 161))
		if !errors.Is(err, domainerrors.ErrInvalidTodoTitle) {
			t.Fatalf("expected ErrInvalidTodoTitle, got %v", err)
		}
	})
}
