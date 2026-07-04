package usecase

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/ports"
	"github.com/endge-lab/service-backend/internal/services"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type CreateTodoInput = adapters.CreateTodoInput
type CreateTodoOutput = adapters.CreateTodoOutput
type CreateTodoUseCase = adapters.CreateTodoService

type createTodoUseCase struct {
	observedUseCase
	txManager      ports.TxManager
	todoRepository ports.TodoRepository
	todoFactory    services.TodoFactory
}

type CreateTodoParams struct {
	TxManager      ports.TxManager
	TodoRepository ports.TodoRepository
	TodoFactory    services.TodoFactory
	Tracer         trace.Tracer
	Logger         *zap.Logger
	Metrics        *UseCaseMetrics
}

func NewCreateTodoUseCase(
	txManager ports.TxManager,
	todoRepository ports.TodoRepository,
	todoFactory services.TodoFactory,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *UseCaseMetrics,
) adapters.CreateTodoService {
	return newCreateTodoUseCase(CreateTodoParams{
		TxManager:      txManager,
		TodoRepository: todoRepository,
		TodoFactory:    todoFactory,
		Tracer:         tracer,
		Logger:         logger,
		Metrics:        metrics,
	})
}

// newCreateTodoUseCase собирает use case создания todo с telemetry и зависимостями.
func newCreateTodoUseCase(params CreateTodoParams) adapters.CreateTodoService {
	return &createTodoUseCase{
		observedUseCase: newObservedUseCase(
			params.Tracer,
			params.Logger.With(zap.String("component", "usecase"), zap.String("usecase", "create_todo")),
			params.Metrics,
		),
		txManager:      params.TxManager,
		todoRepository: params.TodoRepository,
		todoFactory:    params.TodoFactory,
	}
}

// Execute валидирует вход, создаёт todo и сохраняет её в репозитории.
func (u *createTodoUseCase) Execute(ctx context.Context, input adapters.CreateTodoInput) (output *adapters.CreateTodoOutput, err error) {
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

		createdTodo, err = u.todoRepository.Create(txCtx, todo)
		return err
	})
	if err != nil {
		return nil, err
	}

	logger.Debug("create todo use case completed", zap.String("todo_id", createdTodo.ID))

	return &adapters.CreateTodoOutput{
		Todo: createdTodo,
	}, nil
}
