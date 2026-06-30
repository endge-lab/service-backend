package postgres

import (
	"context"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"github.com/endge-lab/service-template-go/internal/util"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TodoRepository сохраняет todo в PostgreSQL.
type TodoRepository struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
	logger *zap.Logger
}

// NewTodoRepository создаёт postgres-адаптер для работы с todo.
func NewTodoRepository(pool *pgxpool.Pool, tracer trace.Tracer, logger *zap.Logger) *TodoRepository {
	return &TodoRepository{
		pool:   pool,
		tracer: tracer,
		logger: logger.With(zap.String("component", "repo"), zap.String("repository", "todo")),
	}
}

// CreateTodo вставляет новую todo и возвращает состояние после сохранения.
func (r *TodoRepository) CreateTodo(ctx context.Context, todo *entities.Todo) (created *entities.Todo, err error) {
	ctx, step := util.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.todo.create",
		attribute.String("repository", "todo"),
	)
	defer func() {
		step.EndTrace(err)
	}()

	logger := util.LoggerWithTrace(ctx, r.logger)
	if todo == nil {
		return nil, domainerrors.ErrInvalidInput
	}

	logger.Debug("creating todo in postgres", zap.String("todo_id", todo.ID))

	row := queryRowerFromContext(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO todos (
			id, title, is_completed, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, is_completed, created_at, updated_at
	`,
		todo.ID,
		todo.Title,
		todo.IsCompleted,
		todo.CreatedAt,
		todo.UpdatedAt,
	)

	created = &entities.Todo{}
	if err = row.Scan(
		&created.ID,
		&created.Title,
		&created.IsCompleted,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return nil, err
	}

	logger.Debug("todo persisted in postgres", zap.String("todo_id", created.ID))
	return created, nil
}
