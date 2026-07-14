package entities

import (
	"time"

	"github.com/google/uuid"
)

type Converter struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	FolderID  uuid.UUID

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

func (Converter) FolderEntityType() FolderEntityType {
	return FolderEntityTypeConverters
}
