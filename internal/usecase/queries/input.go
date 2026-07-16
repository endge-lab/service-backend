package queries

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
