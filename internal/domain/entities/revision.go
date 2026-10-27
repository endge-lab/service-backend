package entities

import (
	"encoding/json"
	"time"
)

type Revision struct {
	ID                     string          `json:"id"`
	WorkspaceID            string          `json:"workspaceId,omitempty"`
	DocumentType           string          `json:"documentType"`
	DocumentID             string          `json:"documentId"`
	DocumentIdentity       string          `json:"documentIdentity"`
	RevisionNumber         int             `json:"revisionNumber"`
	WorkspaceSequence      *int64          `json:"workspaceSequence,omitempty"`
	Operation              string          `json:"operation"`
	ParentRevisionID       *string         `json:"parentRevisionId,omitempty"`
	RestoredFromRevisionID *string         `json:"restoredFromRevisionId,omitempty"`
	CommittedInCommitID    *string         `json:"committedInCommitId,omitempty"`
	MutationBatchID        string          `json:"mutationBatchId"`
	SnapshotVersion        int             `json:"snapshotVersion"`
	Snapshot               json.RawMessage `json:"snapshot"`
	Checksum               string          `json:"checksum"`
	CreatedBy              Actor           `json:"createdBy"`
	// AccessTokenID отмечает изменение агента: CreatedBy остаётся владельцем токена.
	AccessTokenID *string `json:"accessTokenId,omitempty"`
	// AccessTokenName — подпись токена на момент чтения; пустая, если токена нет или он удалён.
	AccessTokenName    *string   `json:"accessTokenName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	ContributorUserIDs []string  `json:"-"`
	Contributors       []Actor   `json:"contributors,omitempty"`
}
