package entities

import (
	"time"

	"github.com/google/uuid"
)

type FolderEntityType string

const (
	FolderEntityTypeComponents FolderEntityType = "components"
	FolderEntityTypeConverters FolderEntityType = "converters"
	FolderEntityTypeQueries    FolderEntityType = "queries"
	FolderEntityTypeDataViews  FolderEntityType = "data-views"
)

type Folder struct {
	ID          uuid.UUID        `json:"id"`
	ProjectID   *uuid.UUID       `json:"project_id,omitempty"`
	EntityType  FolderEntityType `json:"entity_type"`
	Identity    string           `json:"identity"`
	DisplayName string           `json:"display_name"`
	Description *string          `json:"description,omitempty"`
	ParentID    *uuid.UUID       `json:"parent_id,omitempty"`
	IsRoot      bool             `json:"is_root"`
	IsSystem    bool             `json:"is_system"`
	DeletedAt   *time.Time       `json:"deleted_at,omitempty"`
	Meta        map[string]any   `json:"meta"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
