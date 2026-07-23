package query

import (
	"time"

	"github.com/google/uuid"
)

type CreateQueryRequest struct {
	FolderIdentity  string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-queries"`
	Identity        string         `json:"identity" validate:"required,min=1,max=160" example:"example-users-list"`
	DisplayName     string         `json:"displayName" validate:"required,min=1,max=255" example:"Example users list"`
	Description     *string        `json:"description" example:"Loads active users"`
	QueryType       string         `json:"queryType" validate:"required,min=1,max=160" example:"http"`
	Source          map[string]any `json:"source" validate:"required"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs" validate:"omitempty,gt=0" minimum:"1" example:"5000"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled" example:"false"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active" example:"true"`
}

type UpdateQueryRequest struct {
	FolderIdentity  string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-queries"`
	DisplayName     string         `json:"displayName" validate:"required,min=1,max=255" example:"Users list"`
	Description     *string        `json:"description" example:"Loads active users"`
	QueryType       string         `json:"queryType" validate:"required,min=1,max=160" example:"http"`
	Source          map[string]any `json:"source" validate:"required"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs" validate:"omitempty,gt=0" minimum:"1" example:"5000"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled" example:"false"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active" example:"true"`
}

type QueryResponse struct {
	ID              uuid.UUID      `json:"id" example:"00000000-0000-4000-8000-000000000031"`
	ProjectIdentity string         `json:"projectIdentity" example:"demo-project"`
	FolderIdentity  string         `json:"folderIdentity" example:"shared-queries"`
	Identity        string         `json:"identity" example:"users-list"`
	DisplayName     string         `json:"displayName" example:"Users list"`
	Description     *string        `json:"description,omitempty" example:"Loads active users"`
	QueryType       string         `json:"queryType" example:"http"`
	Source          map[string]any `json:"source"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs" example:"5000"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled" example:"false"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active" example:"true"`
	DeletedAt       *time.Time     `json:"deletedAt" example:"2026-07-23T10:00:00Z"`
	CreatedAt       time.Time      `json:"createdAt" example:"2026-07-23T10:00:00Z"`
	UpdatedAt       time.Time      `json:"updatedAt" example:"2026-07-23T10:00:00Z"`
}

type QueriesListResponse struct {
	Items []*QueryResponse `json:"items"`
}
