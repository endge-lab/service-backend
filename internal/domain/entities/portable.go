package entities

import (
	"encoding/json"
	"time"
)

type PortableBundle struct {
	Kind                  string                      `json:"kind"`
	SchemaVersion         int                         `json:"schemaVersion"`
	Workspace             map[string]any              `json:"workspace"`
	Documents             map[string][]map[string]any `json:"documents"`
	InstalledIntegrations []map[string]any            `json:"installedIntegrations"`
}

type ImportPlan struct {
	ID                   string         `json:"planId,omitempty"`
	Valid                bool           `json:"valid"`
	SnapshotChecksum     string         `json:"snapshotChecksum,omitempty"`
	TargetWorkspace      string         `json:"targetWorkspace,omitempty"`
	TargetETag           string         `json:"targetETag,omitempty"`
	ExpiresAt            *time.Time     `json:"expiresAt,omitempty"`
	Incoming             SnapshotCounts `json:"incoming,omitempty"`
	MissingIntegrations  []string       `json:"missingIntegrations,omitempty"`
	Unsupported          []string       `json:"unsupportedCollections,omitempty"`
	ValidationErrors     []string       `json:"validationErrors,omitempty"`
	Warnings             []string       `json:"warnings,omitempty"`
	Conflicts            []string       `json:"conflicts,omitempty"`
	Creates              int            `json:"creates,omitempty"`
	Updates              int            `json:"updates,omitempty"`
	Restores             int            `json:"restores,omitempty"`
	Deletes              int            `json:"deletes,omitempty"`
	ExpectedHeadSequence int64          `json:"expectedHeadSequence,omitempty"`
}

type SnapshotCounts struct {
	Documents    int `json:"documents"`
	Integrations int `json:"integrations"`
}

type SnapshotImportPlan struct {
	ID                   string
	WorkspaceID          string
	SnapshotChecksum     string
	Snapshot             json.RawMessage
	ExpectedGeneration   string
	ExpectedHeadSequence int64
	CreatedBy            Actor
	CreatedAt            time.Time
	ExpiresAt            time.Time
	AppliedAt            *time.Time
}

type SnapshotBackup struct {
	ID            string          `json:"id"`
	WorkspaceID   string          `json:"workspaceId"`
	Kind          string          `json:"kind"`
	Description   *string         `json:"description,omitempty"`
	SchemaVersion int             `json:"schemaVersion"`
	Checksum      string          `json:"checksum"`
	SizeBytes     int64           `json:"sizeBytes"`
	Data          json.RawMessage `json:"data,omitempty" swaggerignore:"true"`
	CreatedBy     Actor           `json:"createdBy"`
	CreatedAt     time.Time       `json:"createdAt"`
	ExpiresAt     *time.Time      `json:"expiresAt,omitempty"`
}

type SnapshotImportResult struct {
	WorkspaceIdentity string         `json:"workspace"`
	Imported          SnapshotCounts `json:"imported"`
	Creates           int            `json:"creates"`
	Updates           int            `json:"updates"`
	Restores          int            `json:"restores"`
	Deletes           int            `json:"deletes"`
	CommitID          string         `json:"commitId"`
	ParentCommitID    string         `json:"parentCommitId"`
}
