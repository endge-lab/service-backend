package query

import (
	"time"

	"github.com/google/uuid"
)

type CreateQueryRequest struct {
	FolderIdentity  string         `json:"folderIdentity" validate:"required"`
	Identity        string         `json:"identity" validate:"required"`
	DisplayName     string         `json:"displayName" validate:"required"`
	Description     *string        `json:"description"`
	QueryType       string         `json:"queryType" validate:"required"`
	Source          map[string]any `json:"source" validate:"required"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active"`
}

type UpdateQueryRequest struct {
	FolderIdentity  string         `json:"folderIdentity" validate:"required"`
	DisplayName     string         `json:"displayName" validate:"required"`
	Description     *string        `json:"description"`
	QueryType       string         `json:"queryType" validate:"required"`
	Source          map[string]any `json:"source" validate:"required"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active"`
}

type QueryResponse struct {
	ID              uuid.UUID      `json:"id"`
	ProjectIdentity string         `json:"projectIdentity"`
	FolderIdentity  string         `json:"folderIdentity"`
	Identity        string         `json:"identity"`
	DisplayName     string         `json:"displayName"`
	Description     *string        `json:"description,omitempty"`
	QueryType       string         `json:"queryType"`
	Source          map[string]any `json:"source"`
	Params          []any          `json:"params"`
	Headers         map[string]any `json:"headers"`
	Auth            map[string]any `json:"auth"`
	TimeoutMS       *int           `json:"timeoutMs"`
	MockData        map[string]any `json:"mockData"`
	MockDataEnabled bool           `json:"mockDataEnabled"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active"`
	DeletedAt       *time.Time     `json:"deletedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type QueriesListResponse struct {
	Items []*QueryResponse `json:"items"`
}
