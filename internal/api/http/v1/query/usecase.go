package query

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/queries"
)

// UseCase is the application contract consumed by the query HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input queries.CreateQueryInput) (*queries.QueryWithFolder, error)
	Update(ctx context.Context, input queries.UpdateQueryInput) (*queries.QueryWithFolder, error)
	GetByIdentity(ctx context.Context, input queries.GetQueryInput) (*queries.QueryWithFolder, error)
	List(ctx context.Context, input queries.ListQueriesInput) ([]*queries.QueryWithFolder, error)
	SoftDelete(ctx context.Context, input queries.QueryIdentityInput) error
	Restore(ctx context.Context, input queries.QueryIdentityInput) error
	HardDelete(ctx context.Context, input queries.QueryIdentityInput) error
}
