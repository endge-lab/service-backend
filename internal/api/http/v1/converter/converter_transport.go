package converter

import (
	"time"

	"github.com/google/uuid"
)

type CreateConverterRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required"`
	Identity       string         `json:"identity" validate:"required"`
	DisplayName    string         `json:"displayName" validate:"required"`
	Description    *string        `json:"description"`
	ConverterType  string         `json:"converterType" validate:"required"`
	Source         map[string]any `json:"source"`
	IsSystem       bool           `json:"isSystem"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active"`
}
type UpdateConverterRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required"`
	DisplayName    string         `json:"displayName" validate:"required"`
	Description    *string        `json:"description"`
	ConverterType  string         `json:"converterType" validate:"required"`
	Source         map[string]any `json:"source"`
	IsSystem       bool           `json:"isSystem"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active"`
}
type ConverterResponse struct {
	ID              uuid.UUID      `json:"id"`
	ProjectIdentity string         `json:"projectIdentity"`
	FolderIdentity  string         `json:"folderIdentity"`
	Identity        string         `json:"identity"`
	DisplayName     string         `json:"displayName"`
	Description     *string        `json:"description,omitempty"`
	ConverterType   string         `json:"converterType"`
	Source          map[string]any `json:"source"`
	IsSystem        bool           `json:"isSystem"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active"`
	DeletedAt       *time.Time     `json:"deletedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}
type ConvertersListResponse struct {
	Items []*ConverterResponse `json:"items"`
}
