package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/endge-lab/service-template-go/internal/util"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type txContextKey struct{}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TxManager struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
	logger *zap.Logger
}

func NewTxManager(pool *pgxpool.Pool, tracer trace.Tracer, logger *zap.Logger) *TxManager {
	return &TxManager{
		pool:   pool,
		tracer: tracer,
		logger: logger.With(zap.String("component", "repo"), zap.String("repository", "transaction_manager")),
	}
}

func (m *TxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	ctx, step := util.StartTrace(
		ctx,
		m.tracer,
		m.logger,
		"repo.transaction.begin",
		attribute.String("repository", "transaction_manager"),
	)
	defer func() {
		step.EndTrace(err)
	}()

	logger := util.LoggerWithTrace(ctx, m.logger)
	logger.Debug("opening postgres transaction")

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		logger.Warn("rolling back postgres transaction", zap.Error(err))
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return fmt.Errorf("rollback transaction: %v (original error: %w)", rollbackErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logger.Debug("postgres transaction committed")
	return nil
}

func queryRowerFromContext(ctx context.Context, pool *pgxpool.Pool) queryRower {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}

	return pool
}
