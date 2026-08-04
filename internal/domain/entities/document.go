package entities

import (
	"encoding/json"
	"time"
)

type Document struct {
	ID             string          `json:"id"`
	WorkspaceID    string          `json:"-"`
	Type           string          `json:"type"`
	Identity       string          `json:"identity"`
	DisplayName    string          `json:"displayName"`
	Description    *string         `json:"description,omitempty"`
	FolderIdentity *string         `json:"folderIdentity,omitempty"`
	ManagedBy      string          `json:"managedBy"`
	ManagedByID    *string         `json:"managedById,omitempty"`
	Meta           json.RawMessage `json:"meta"`
	Data           json.RawMessage `json:"data"`
	Active         bool            `json:"active"`
	DeletedAt      *time.Time      `json:"deletedAt,omitempty"`
	Revision       int             `json:"revision"`
	CreatedBy      Actor           `json:"createdBy"`
	UpdatedBy      Actor           `json:"updatedBy"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}
