package entities

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID                    uuid.UUID      `json:"id"`
	Identity              string         `json:"identity"`
	DisplayName           string         `json:"display_name"`
	ExtendSettings        bool           `json:"extend_settings"`
	SettingsID            *uuid.UUID     `json:"settings_id,omitempty"`
	NavigationID          *uuid.UUID     `json:"navigation_id,omitempty"`
	FolderID              *uuid.UUID     `json:"folder_id,omitempty"`
	AllowedEnvironmentIDs []uuid.UUID    `json:"allowed_environment_ids"`
	DeletedAt             *time.Time     `json:"deleted_at,omitempty"`
	Meta                  map[string]any `json:"meta"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type CreateProject struct {
	Identity              string
	DisplayName           string
	ExtendSettings        bool
	SettingsID            *uuid.UUID
	NavigationID          *uuid.UUID
	FolderID              *uuid.UUID
	AllowedEnvironmentIDs []uuid.UUID
	Meta                  map[string]any
}

type UpdateProject struct {
	DisplayName           *string
	ExtendSettings        *bool
	SettingsID            *uuid.UUID
	NavigationID          *uuid.UUID
	FolderID              *uuid.UUID
	AllowedEnvironmentIDs []uuid.UUID
	Meta                  map[string]any
}
