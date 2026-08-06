package entities

import (
	"encoding/json"
	"time"
)

type Release struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspaceId"`
	Identity       string    `json:"identity"`
	DisplayName    string    `json:"displayName"`
	Description    *string   `json:"description,omitempty"`
	SourceCommitID string    `json:"sourceCommitId"`
	HeadSequence   int64     `json:"headSequence"`
	SchemaVersion  int       `json:"schemaVersion"`
	Checksum       string    `json:"checksum"`
	CreatedBy      Actor     `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ReleaseArtifact содержит неизменяемый переносимый JSON релиза отдельно от
// его metadata. Это не позволяет list/get операций случайно читать большой
// artifact из постоянного хранилища.
type ReleaseArtifact struct {
	ReleaseID   string          `json:"releaseId"`
	WorkspaceID string          `json:"workspaceId"`
	Identity    string          `json:"identity"`
	Checksum    string          `json:"checksum"`
	Data        json.RawMessage `json:"data"`
}
