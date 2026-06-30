package usecase

import (
	"context"
	"strings"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
	"github.com/endge-lab/service-template-go/internal/ports"
	"github.com/endge-lab/service-template-go/internal/services"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// CreateTodoInput описывает вход для сценария создания todo.
type CreateTodoInput struct {
	Title string
}

// CreateTodoOutput возвращает созданную доменную todo.
type CreateTodoOutput struct {
	Todo *entities.Todo
}

// CreateTodoUseCase создаёт todo внутри транзакционной границы use case.
type CreateTodoUseCase interface {
	Execute(ctx context.Context, input CreateTodoInput) (*CreateTodoOutput, error)
}

type createTodoUseCase struct {
	observedUseCase
	txManager      ports.TxManager
	todoRepository ports.TodoRepository
	todoFactory    services.TodoFactory
}

// NewCreateTodoUseCase собирает use case создания todo с telemetry и зависимостями.
func NewCreateTodoUseCase(
	txManager ports.TxManager,
	todoRepository ports.TodoRepository,
	todoFactory services.TodoFactory,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *UseCaseMetrics,
) CreateTodoUseCase {
	return &createTodoUseCase{
		observedUseCase: newObservedUseCase(
			tracer,
			logger.With(zap.String("component", "usecase"), zap.String("usecase", "create_todo")),
			metrics,
		),
		txManager:      txManager,
		todoRepository: todoRepository,
		todoFactory:    todoFactory,
	}
}

// Execute валидирует вход, создаёт todo и сохраняет её в репозитории.
func (u *createTodoUseCase) Execute(ctx context.Context, input CreateTodoInput) (output *CreateTodoOutput, err error) {
	title := strings.TrimSpace(input.Title)
	ctx, obs := u.startObservedOperation(ctx, "create_todo", []attribute.KeyValue{
		attribute.Int("todo.title_length", len(title)),
	}, nil)
	defer obs.End(&err)

	logger := obs.Logger()
	logger.Debug("create todo use case started", zap.Int("title_length", len(title)))

	var createdTodo *entities.Todo

	err = u.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		todo, err := u.todoFactory.New(txCtx, input.Title)
		if err != nil {
			return err
		}

		createdTodo, err = u.todoRepository.CreateTodo(txCtx, todo)
		return err
	})
	if err != nil {
		return nil, err
	}

	logger.Debug("create todo use case completed", zap.String("todo_id", createdTodo.ID))

	return &CreateTodoOutput{
		Todo: createdTodo,
	}, nil
}
