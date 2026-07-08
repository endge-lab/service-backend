package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type baseRepository struct {
	q      *sqlc.Queries
	tracer trace.Tracer
	logger *zap.Logger
}

func newBaseRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger, repository string) *baseRepository {
	return &baseRepository{
		q:      queries,
		tracer: tracer,
		logger: logger.With(zap.String("component", "repo"), zap.String("repository", repository)),
	}
}

func (r *baseRepository) queries(ctx context.Context) *sqlc.Queries {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}

	return r.q
}
