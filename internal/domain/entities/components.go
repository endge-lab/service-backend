package entities

import (
	"time"

	"github.com/google/uuid"
)

type ComponentType string

const (
	ComponentTypeSFC ComponentType = "component-sfc"
)

type ComponentSourceFormat string

const (
	ComponentSourceFormatSFC ComponentSourceFormat = "sfc"
)

type Component struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	FolderID  uuid.UUID

	Identity    string
	DisplayName string
	Description *string

	ComponentType ComponentType
	Source        string
	SourceFormat  ComponentSourceFormat

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any

	Active    bool
	DeletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Component) FolderEntityType() FolderEntityType {
	return FolderEntityTypeComponents
}
