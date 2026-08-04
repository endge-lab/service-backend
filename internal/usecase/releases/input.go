package releases

// CreateInput содержит данные immutable release.
type CreateInput struct {
	Identity       string
	DisplayName    string
	Description    *string
	SourceCommitID string
}

// values декодирует входные данные в карту значений.
func (i CreateInput) values() map[string]any {
	values := map[string]any{
		"identity":       i.Identity,
		"displayName":    i.DisplayName,
		"sourceCommitId": i.SourceCommitID,
	}
	if i.Description != nil {
		values["description"] = *i.Description
	}
	return values
}
