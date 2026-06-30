package services

import (
	"context"
	"time"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
	"github.com/endge-lab/service-template-go/internal/domain/valueobjects"
	"github.com/endge-lab/service-template-go/internal/util"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TodoFactory собирает валидную доменную todo из входных данных.
type TodoFactory interface {
	New(ctx context.Context, title string) (*entities.Todo, error)
}

type todoFactory struct {
	tracer trace.Tracer
	logger *zap.Logger
}

// NewTodoFactory создаёт сервис, который нормализует и конструирует todo.
func NewTodoFactory(tracer trace.Tracer, logger *zap.Logger) TodoFactory {
	return &todoFactory{
		tracer: tracer,
		logger: logger.With(zap.String("component", "service"), zap.String("service", "todo_factory")),
	}
}

// New создаёт новую todo и заполняет технические поля по правилам домена.
func (f *todoFactory) New(ctx context.Context, title string) (todo *entities.Todo, err error) {
	normalizedTitle := title
	ctx, step := util.StartTrace(
		ctx,
		f.tracer,
		f.logger,
		"service.todo_factory.new",
		attribute.String("service", "todo_factory"),
		attribute.Int("todo.title_length", len(normalizedTitle)),
	)
	defer func() {
		step.EndTrace(err)
	}()

	logger := util.LoggerWithTrace(ctx, f.logger)
	logger.Debug("todo factory started", zap.Int("title_length", len(normalizedTitle)))

	todoTitle, err := valueobjects.NewTodoTitle(title)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	todo = &entities.Todo{
		ID:          uuid.NewString(),
		Title:       todoTitle.Value(),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	logger.Debug("todo factory completed", zap.String("todo_id", todo.ID))
	return todo, nil
}
