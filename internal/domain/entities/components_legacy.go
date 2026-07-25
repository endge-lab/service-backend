package entities

import (
	"time"

	"github.com/google/uuid"
)

type RComponentLegacyType string

const (
	RComponentLegacyTypeSFC RComponentLegacyType = "component-sfc"
)

type RComponentLegacySourceFormat string

const (
	RComponentLegacySourceFormatSFC RComponentLegacySourceFormat = "sfc"
)

type RComponentLegacy struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	FolderID    uuid.UUID

	Identity    string
	DisplayName string
	Description *string

	ComponentType RComponentLegacyType
	Source        string
	SourceFormat  RComponentLegacySourceFormat

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any

	Active    bool
	DeletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (RComponentLegacy) FolderEntityType() FolderEntityType {
	return FolderEntityTypeComponentsLegacy
}
