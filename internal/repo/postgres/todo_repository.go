package postgres

import (
	"context"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TodoRepository saves todos in PostgreSQL through sqlc.
type TodoRepository struct {
	*baseRepository
	tracer trace.Tracer
	logger *zap.Logger
}

func NewTodoRepository(_ *pgxpool.Pool, queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *TodoRepository {
	return &TodoRepository{
		baseRepository: newBaseRepository(queries),
		tracer:         tracer,
		logger:         logger.With(zap.String("component", "repo"), zap.String("repository", "todo")),
	}
}

func (r *TodoRepository) Create(ctx context.Context, todo *entities.Todo) (created *entities.Todo, err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.todo.create",
		attribute.String("repository", "todo"),
	)
	defer func() {
		step.End(err)
	}()

	logger := logging.WithContext(ctx, r.logger)
	if todo == nil {
		return nil, domainerrors.ErrInvalidInput
	}

	id, err := parseTodoID(todo.ID)
	if err != nil {
		return nil, err
	}

	logger.Debug("creating todo in postgres", zap.String("todo_id", todo.ID))

	record, err := r.queries(ctx).CreateTodo(ctx, sqlc.CreateTodoParams{
		ID:          id,
		Title:       todo.Title,
		IsCompleted: todo.IsCompleted,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		return nil, mapPostgresError(err, "todo.create")
	}

	created = mapTodo(record)
	logger.Debug("todo persisted in postgres", zap.String("todo_id", created.ID))
	return created, nil
}

func (r *TodoRepository) GetByID(ctx context.Context, id string) (todo *entities.Todo, err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.todo.get_by_id",
		attribute.String("repository", "todo"),
		attribute.String("todo.id", id),
	)
	defer func() {
		step.End(err)
	}()

	todoID, err := parseTodoID(id)
	if err != nil {
		return nil, err
	}

	record, err := r.queries(ctx).GetTodoByID(ctx, todoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrTodoNotFound
		}
		return nil, mapPostgresError(err, "todo.get_by_id")
	}

	return mapTodo(record), nil
}

func (r *TodoRepository) Update(ctx context.Context, todo *entities.Todo) (updated *entities.Todo, err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.todo.update",
		attribute.String("repository", "todo"),
	)
	defer func() {
		step.End(err)
	}()

	if todo == nil {
		return nil, domainerrors.ErrInvalidInput
	}

	id, err := parseTodoID(todo.ID)
	if err != nil {
		return nil, err
	}

	record, err := r.queries(ctx).UpdateTodo(ctx, sqlc.UpdateTodoParams{
		ID:          id,
		Title:       todo.Title,
		IsCompleted: todo.IsCompleted,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrTodoNotFound
		}
		return nil, mapPostgresError(err, "todo.update")
	}

	return mapTodo(record), nil
}

func (r *TodoRepository) Delete(ctx context.Context, id string) (err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.todo.delete",
		attribute.String("repository", "todo"),
		attribute.String("todo.id", id),
	)
	defer func() {
		step.End(err)
	}()

	todoID, err := parseTodoID(id)
	if err != nil {
		return err
	}

	rows, err := r.queries(ctx).DeleteTodo(ctx, todoID)
	if err != nil {
		return mapPostgresError(err, "todo.delete")
	}
	if rows == 0 {
		return domainerrors.ErrTodoNotFound
	}

	return nil
}

func parseTodoID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, domainerrors.WithDetails(
			domainerrors.InvalidInput("todo.invalid_id", "Некорректный идентификатор задачи"),
			map[string]any{"id": id},
		)
	}

	return parsed, nil
}
