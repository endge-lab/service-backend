package entities

import (
	"time"

	"github.com/google/uuid"
)

type RDataView struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	FolderID  uuid.UUID
	QueryID   uuid.UUID

	Identity    string
	DisplayName string
	Description *string

	ViewType     string
	Source       map[string]any
	InputSchema  map[string]any
	OutputSchema map[string]any
	Meta         map[string]any
	Active       bool

	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (RDataView) FolderEntityType() FolderEntityType {
	return FolderEntityTypeDataViews
}
