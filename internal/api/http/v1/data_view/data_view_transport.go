package data_view

import (
	"time"

	"github.com/google/uuid"
)

type CreateDataViewRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-data-views"`
	QueryIdentity  string         `json:"queryIdentity" validate:"required,min=1,max=160" example:"users-list"`
	Identity       string         `json:"identity" validate:"required,min=1,max=160" example:"example-users-table"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255" example:"Example users table"`
	Description    *string        `json:"description" example:"Table view for users"`
	ViewType       string         `json:"viewType" validate:"required,min=1,max=160" example:"table"`
	Source         map[string]any `json:"source" validate:"required"`
	InputSchema    map[string]any `json:"inputSchema"`
	OutputSchema   map[string]any `json:"outputSchema"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active" example:"true"`
}

type UpdateDataViewRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-data-views"`
	QueryIdentity  string         `json:"queryIdentity" validate:"required,min=1,max=160" example:"users-list"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255" example:"Users table"`
	Description    *string        `json:"description" example:"Table view for users"`
	ViewType       string         `json:"viewType" validate:"required,min=1,max=160" example:"table"`
	Source         map[string]any `json:"source" validate:"required"`
	InputSchema    map[string]any `json:"inputSchema"`
	OutputSchema   map[string]any `json:"outputSchema"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active" example:"true"`
}

type DataViewResponse struct {
	ID              uuid.UUID      `json:"id" example:"00000000-0000-4000-8000-000000000041"`
	ProjectIdentity string         `json:"projectIdentity" example:"demo-project"`
	FolderIdentity  string         `json:"folderIdentity" example:"shared-data-views"`
	QueryIdentity   string         `json:"queryIdentity" example:"users-list"`
	Identity        string         `json:"identity" example:"users-table"`
	DisplayName     string         `json:"displayName" example:"Users table"`
	Description     *string        `json:"description,omitempty" example:"Table view for users"`
	ViewType        string         `json:"viewType" example:"table"`
	Source          map[string]any `json:"source"`
	InputSchema     map[string]any `json:"inputSchema"`
	OutputSchema    map[string]any `json:"outputSchema"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active" example:"true"`
	DeletedAt       *time.Time     `json:"deletedAt" example:"2026-07-23T10:00:00Z"`
	CreatedAt       time.Time      `json:"createdAt" example:"2026-07-23T10:00:00Z"`
	UpdatedAt       time.Time      `json:"updatedAt" example:"2026-07-23T10:00:00Z"`
}

type DataViewsListResponse struct {
	Items []*DataViewResponse `json:"items"`
}
