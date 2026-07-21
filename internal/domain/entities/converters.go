package entities

import (
	"time"

	"github.com/google/uuid"
)

type RConverter struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	FolderID    uuid.UUID

	Identity    string
	DisplayName string
	Description *string

	ConverterType string
	Source        map[string]any

	IsSystem bool
	Meta     map[string]any
	Active   bool

	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (RConverter) FolderEntityType() FolderEntityType {
	return FolderEntityTypeConverters
}
