package entities

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          uuid.UUID      `json:"id"`
	Identity    string         `json:"identity"`
	DisplayName string         `json:"display_name"`
	Description *string        `json:"description,omitempty"`
	Active      bool           `json:"active"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
	Meta        map[string]any `json:"meta"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateProject struct {
	Identity    string
	DisplayName string
	Description *string
	Active      bool
	Meta        map[string]any
}

type UpdateProject struct {
	DisplayName *string
	Description *string
	Active      *bool
	Meta        map[string]any
}
