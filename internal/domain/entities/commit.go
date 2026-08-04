package entities

import "time"

type Commit struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspaceId"`
	ParentCommitID *string        `json:"parentCommitId,omitempty"`
	BaseSequence   int64          `json:"baseSequence"`
	HeadSequence   int64          `json:"headSequence"`
	Message        string         `json:"message"`
	RevisionPolicy string         `json:"revisionPolicy"`
	Operation      string         `json:"operation"`
	CreatedBy      Actor          `json:"createdBy"`
	CreatedAt      time.Time      `json:"createdAt"`
	Changes        []CommitChange `json:"changes,omitempty"`
}

type CommitChange struct {
	DocumentType     string  `json:"documentType"`
	DocumentID       string  `json:"documentId"`
	BeforeRevisionID *string `json:"beforeRevisionId,omitempty"`
	AfterRevisionID  *string `json:"afterRevisionId,omitempty"`
	Operation        string  `json:"operation"`
}
