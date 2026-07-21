package entities

import (
	"time"

	"github.com/google/uuid"
)

type RQuery struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	FolderID    uuid.UUID

	Identity    string
	DisplayName string
	Description *string

	QueryType string
	Source    map[string]any
	Params    []any
	Headers   map[string]any
	Auth      map[string]any
	TimeoutMS *int
	MockData  map[string]any

	MockDataEnabled bool
	Meta            map[string]any
	Active          bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (RQuery) FolderEntityType() FolderEntityType {
	return FolderEntityTypeQueries
}
