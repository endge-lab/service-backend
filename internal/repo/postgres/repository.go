package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"

	"go.uber.org/zap"
)

type baseRepository struct {
	q        *sqlc.Queries
	observer observability.Observer
}

func newBaseRepository(queries *sqlc.Queries, core *observability.Core, metrics *RepositoryMetrics, repository string) *baseRepository {
	observer := core.For(observability.LayerRepository, "postgres_"+repository+"_repository").WithRecorder(metrics).WithFields(zap.String("repository", repository))
	return &baseRepository{
		q:        queries,
		observer: observer,
	}
}

func (r *baseRepository) queries(ctx context.Context) *sqlc.Queries {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}

	return r.q
}
