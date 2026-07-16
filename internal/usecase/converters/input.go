package converters

type CreateConverterInput struct {
	ProjectIdentity string
	FolderIdentity  string

	Identity      string
	DisplayName   string
	Description   *string
	ConverterType string
	Source        map[string]any

	IsSystem bool
	Meta     map[string]any
	Active   bool
}

type UpdateConverterInput struct {
	ProjectIdentity   string
	ConverterIdentity string
	FolderIdentity    string

	DisplayName   string
	Description   *string
	ConverterType string
	Source        map[string]any

	IsSystem bool
	Meta     map[string]any
	Active   bool
}

type GetConverterInput struct {
	ProjectIdentity   string
	ConverterIdentity string
}

type ConverterIdentityInput struct {
	ProjectIdentity   string
	ConverterIdentity string
}

type ListConvertersInput struct {
	ProjectIdentity string
	FolderIdentity  *string
}
