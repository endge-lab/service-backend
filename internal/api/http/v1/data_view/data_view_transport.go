package data_view

import (
	"time"

	"github.com/google/uuid"
)

type CreateDataViewRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160"`
	QueryIdentity  string         `json:"queryIdentity" validate:"required,min=1,max=160"`
	Identity       string         `json:"identity" validate:"required,min=1,max=160"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255"`
	Description    *string        `json:"description"`
	ViewType       string         `json:"viewType" validate:"required,min=1,max=160"`
	Source         map[string]any `json:"source" validate:"required"`
	InputSchema    map[string]any `json:"inputSchema"`
	OutputSchema   map[string]any `json:"outputSchema"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active"`
}

type UpdateDataViewRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160"`
	QueryIdentity  string         `json:"queryIdentity" validate:"required,min=1,max=160"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255"`
	Description    *string        `json:"description"`
	ViewType       string         `json:"viewType" validate:"required,min=1,max=160"`
	Source         map[string]any `json:"source" validate:"required"`
	InputSchema    map[string]any `json:"inputSchema"`
	OutputSchema   map[string]any `json:"outputSchema"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active"`
}

type DataViewResponse struct {
	ID              uuid.UUID      `json:"id"`
	ProjectIdentity string         `json:"projectIdentity"`
	FolderIdentity  string         `json:"folderIdentity"`
	QueryIdentity   string         `json:"queryIdentity"`
	Identity        string         `json:"identity"`
	DisplayName     string         `json:"displayName"`
	Description     *string        `json:"description,omitempty"`
	ViewType        string         `json:"viewType"`
	Source          map[string]any `json:"source"`
	InputSchema     map[string]any `json:"inputSchema"`
	OutputSchema    map[string]any `json:"outputSchema"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active"`
	DeletedAt       *time.Time     `json:"deletedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type DataViewsListResponse struct {
	Items []*DataViewResponse `json:"items"`
}
