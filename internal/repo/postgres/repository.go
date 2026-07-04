package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

type baseRepository struct {
	q *sqlc.Queries
}

func newBaseRepository(queries *sqlc.Queries) *baseRepository {
	return &baseRepository{
		q: queries,
	}
}

func (r *baseRepository) queries(ctx context.Context) *sqlc.Queries {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}

	return r.q
}
