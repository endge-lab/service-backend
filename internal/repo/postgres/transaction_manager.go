package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type txContextKey struct{}

type TxManager struct {
	pool     *pgxpool.Pool
	observer observability.Observer
}

func NewTxManager(pool *pgxpool.Pool, core *observability.Core, metrics *RepositoryMetrics) *TxManager {
	observer := core.For(observability.LayerRepository, "postgres_transaction_manager").WithRecorder(metrics).WithFields(zap.String("repository", "transaction_manager"))
	return &TxManager{
		pool:     pool,
		observer: observer,
	}
}

// WithinTransaction выполняет callback внутри транзакции.
//
// Параметры:
//
//	ctx - контекст выполнения
//	fn - функция, которая получает транзакционный context
//
// Что делает функция:
//
//	Открывает PostgreSQL transaction и передает ее через context.
//	Выполняет commit при успехе или rollback при ошибке callback.
//
// Возвращаемые значения:
//
//	error - ошибка callback, открытия, commit или rollback транзакции
func (m *TxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		m.observer.Tracer(),
		m.observer.Logger(),
		"repo.transaction.begin",
		attribute.String("repository", "transaction_manager"),
	)
	defer func() {
		step.End(err)
	}()

	logger := logging.WithContext(ctx, m.observer.Logger())
	logger.Debug("opening postgres transaction")

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		logger.Warn("rolling back postgres transaction", zap.Error(err))
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return fmt.Errorf("rollback transaction: %v (original transport: %w)", rollbackErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logger.Debug("postgres transaction committed")
	return nil
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok && tx != nil {
		return tx, true
	}

	return nil, false
}
