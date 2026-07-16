package data_views

type CreateDataViewInput struct {
	ProjectIdentity string
	FolderIdentity  string
	QueryIdentity   string

	Identity    string
	DisplayName string
	Description *string

	ViewType     string
	Source       map[string]any
	InputSchema  map[string]any
	OutputSchema map[string]any
	Meta         map[string]any
	Active       bool
}

type UpdateDataViewInput struct {
	ProjectIdentity  string
	DataViewIdentity string
	FolderIdentity   string
	QueryIdentity    string

	DisplayName string
	Description *string

	ViewType     string
	Source       map[string]any
	InputSchema  map[string]any
	OutputSchema map[string]any
	Meta         map[string]any
	Active       bool
}

type GetDataViewInput struct {
	ProjectIdentity  string
	DataViewIdentity string
}

type DataViewIdentityInput struct {
	ProjectIdentity  string
	DataViewIdentity string
}

type ListDataViewsInput struct {
	ProjectIdentity string
	FolderIdentity  *string
	QueryIdentity   *string
}
