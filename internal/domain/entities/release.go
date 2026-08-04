package entities

import (
	"encoding/json"
	"time"
)

type Release struct {
	ID             string          `json:"id"`
	WorkspaceID    string          `json:"workspaceId"`
	Identity       string          `json:"identity"`
	DisplayName    string          `json:"displayName"`
	Description    *string         `json:"description,omitempty"`
	SourceCommitID string          `json:"sourceCommitId"`
	HeadSequence   int64           `json:"headSequence"`
	SchemaVersion  int             `json:"schemaVersion"`
	Checksum       string          `json:"checksum"`
	Data           json.RawMessage `json:"data,omitempty"`
	CreatedBy      Actor           `json:"createdBy"`
	CreatedAt      time.Time       `json:"createdAt"`
}
