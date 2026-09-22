package releases

// CreateInput содержит данные immutable release.
type CreateInput struct {
	Identity       string
	DisplayName    string
	Description    *string
	SourceCommitID string
}
