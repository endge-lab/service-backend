package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateQueryInput struct {
	ProjectIdentity string
	FolderIdentity  string

	Identity    string
	DisplayName string
	Description *string

	QueryType string
	Source    map[string]any
	Params    []any
	Headers   map[string]any
	Auth      map[string]any
	TimeoutMS *int
	MockData  map[string]any

	MockDataEnabled bool
	Meta            map[string]any
	Active          bool
}

type UpdateQueryInput struct {
	ProjectIdentity string
	QueryIdentity   string
	FolderIdentity  string

	DisplayName string
	Description *string

	QueryType string
	Source    map[string]any
	Params    []any
	Headers   map[string]any
	Auth      map[string]any
	TimeoutMS *int
	MockData  map[string]any

	MockDataEnabled bool
	Meta            map[string]any
	Active          bool
}

type GetQueryInput struct {
	ProjectIdentity string
	QueryIdentity   string
}

type QueryIdentityInput struct {
	ProjectIdentity string
	QueryIdentity   string
}

type ListQueriesInput struct {
	ProjectIdentity string
	FolderIdentity  *string
	QueryType       *string
}

type QueryWithFolder struct {
	Query          *entities.Query
	FolderIdentity string
}

type QueryService interface {
	Create(ctx context.Context, input CreateQueryInput) (*QueryWithFolder, error)
	Update(ctx context.Context, input UpdateQueryInput) (*QueryWithFolder, error)
	GetByIdentity(ctx context.Context, input GetQueryInput) (*QueryWithFolder, error)
	List(ctx context.Context, input ListQueriesInput) ([]*QueryWithFolder, error)
	SoftDelete(ctx context.Context, input QueryIdentityInput) error
	Restore(ctx context.Context, input QueryIdentityInput) error
	HardDelete(ctx context.Context, input QueryIdentityInput) error
	Count(ctx context.Context, input ListQueriesInput) (int64, error)
}
