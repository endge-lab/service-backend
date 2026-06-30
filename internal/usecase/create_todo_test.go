package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	domainentities "github.com/endge-lab/service-template-go/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type fakeTxManager struct {
	calls int
}

func (m *fakeTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type fakeTodoRepository struct {
	received *domainentities.Todo
	result   *domainentities.Todo
	err      error
}

func (r *fakeTodoRepository) CreateTodo(_ context.Context, todo *domainentities.Todo) (*domainentities.Todo, error) {
	r.received = todo
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

type fakeTodoFactory struct {
	result *domainentities.Todo
	err    error
}

func (f *fakeTodoFactory) New(_ context.Context, _ string) (*domainentities.Todo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestCreateTodoUseCaseExecute(t *testing.T) {
	ctx := context.Background()
	metrics, err := NewUseCaseMetrics(otel.GetMeterProvider().Meter("test"))
	if err != nil {
		t.Fatalf("expected metrics to initialize, got %v", err)
	}
	logger := zap.NewNop()
	tracer := otel.Tracer("test")

	t.Run("creates todo inside transaction", func(t *testing.T) {
		txManager := &fakeTxManager{}
		now := time.Now().UTC()
		todo := &domainentities.Todo{
			ID:          "todo-1",
			Title:       "Pay invoice",
			IsCompleted: false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		repository := &fakeTodoRepository{result: todo}
		factory := &fakeTodoFactory{result: todo}
		useCase := NewCreateTodoUseCase(txManager, repository, factory, tracer, logger, metrics)

		output, err := useCase.Execute(ctx, CreateTodoInput{Title: "Pay invoice"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if txManager.calls != 1 {
			t.Fatalf("expected transaction manager to be called once, got %d", txManager.calls)
		}
		if repository.received == nil {
			t.Fatal("expected repository to receive todo")
		}
		if output == nil || output.Todo == nil {
			t.Fatal("expected todo in output")
		}
		if output.Todo.ID != "todo-1" {
			t.Fatalf("expected todo id todo-1, got %q", output.Todo.ID)
		}
	})

	t.Run("returns factory validation error", func(t *testing.T) {
		txManager := &fakeTxManager{}
		repository := &fakeTodoRepository{}
		factory := &fakeTodoFactory{err: domainerrors.ErrInvalidTodoTitle}
		useCase := NewCreateTodoUseCase(txManager, repository, factory, tracer, logger, metrics)

		_, err := useCase.Execute(ctx, CreateTodoInput{Title: ""})
		if !errors.Is(err, domainerrors.ErrInvalidTodoTitle) {
			t.Fatalf("expected ErrInvalidTodoTitle, got %v", err)
		}
		if repository.received != nil {
			t.Fatal("expected repository not to be called when factory fails")
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		txManager := &fakeTxManager{}
		now := time.Now().UTC()
		todo := &domainentities.Todo{
			ID:          "todo-2",
			Title:       "Book appointment",
			CreatedAt:   now,
			UpdatedAt:   now,
			IsCompleted: false,
		}
		repository := &fakeTodoRepository{err: domainerrors.ErrConflict}
		factory := &fakeTodoFactory{result: todo}
		useCase := NewCreateTodoUseCase(txManager, repository, factory, tracer, logger, metrics)

		_, err := useCase.Execute(ctx, CreateTodoInput{Title: todo.Title})
		if !errors.Is(err, domainerrors.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})
}
