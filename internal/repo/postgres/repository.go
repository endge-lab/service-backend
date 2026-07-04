package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type baseRepository struct {
	q *sqlc.Queries
}

func newBaseRepository(pool *pgxpool.Pool) baseRepository {
	return baseRepository{
		q: sqlc.New(pool),
	}
}

func (r baseRepository) queries(ctx context.Context) *sqlc.Queries {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}

	return r.q
}
