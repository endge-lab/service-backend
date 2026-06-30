package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestTodoFactoryNew(t *testing.T) {
	factory := NewTodoFactory(otel.Tracer("test"), zap.NewNop())

	t.Run("creates normalized todo", func(t *testing.T) {
		todo, err := factory.New(context.Background(), "  Call clinic  ")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if todo == nil {
			t.Fatal("expected todo to be created")
		}
		if todo.ID == "" {
			t.Fatal("expected todo id to be generated")
		}
		if todo.Title != "Call clinic" {
			t.Fatalf("expected normalized title, got %q", todo.Title)
		}
		if todo.IsCompleted {
			t.Fatal("expected new todo to be incomplete")
		}
		if todo.CreatedAt.IsZero() || todo.UpdatedAt.IsZero() {
			t.Fatal("expected timestamps to be set")
		}
		if todo.CreatedAt.Location() != time.UTC || todo.UpdatedAt.Location() != time.UTC {
			t.Fatal("expected UTC timestamps")
		}
	})

	t.Run("rejects invalid title", func(t *testing.T) {
		_, err := factory.New(context.Background(), "")
		if !errors.Is(err, domainerrors.ErrInvalidTodoTitle) {
			t.Fatalf("expected ErrInvalidTodoTitle, got %v", err)
		}
	})
}
