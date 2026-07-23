package converter

import (
	"time"

	"github.com/google/uuid"
)

type CreateConverterRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-converters"`
	Identity       string         `json:"identity" validate:"required,min=1,max=160" example:"example-date-to-string"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255" example:"Example date to string"`
	Description    *string        `json:"description" example:"Formats an ISO date"`
	ConverterType  string         `json:"converterType" validate:"required,min=1,max=160" example:"javascript"`
	Source         map[string]any `json:"source"`
	IsSystem       bool           `json:"isSystem" example:"false"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active" example:"true"`
}
type UpdateConverterRequest struct {
	FolderIdentity string         `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-converters"`
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255" example:"Date to string"`
	Description    *string        `json:"description" example:"Formats an ISO date"`
	ConverterType  string         `json:"converterType" validate:"required,min=1,max=160" example:"javascript"`
	Source         map[string]any `json:"source"`
	IsSystem       bool           `json:"isSystem" example:"false"`
	Meta           map[string]any `json:"meta"`
	Active         bool           `json:"active" example:"true"`
}
type ConverterResponse struct {
	ID              uuid.UUID      `json:"id" example:"00000000-0000-4000-8000-000000000021"`
	ProjectIdentity string         `json:"projectIdentity" example:"demo-project"`
	FolderIdentity  string         `json:"folderIdentity" example:"shared-converters"`
	Identity        string         `json:"identity" example:"date-to-string"`
	DisplayName     string         `json:"displayName" example:"Date to string"`
	Description     *string        `json:"description,omitempty" example:"Formats an ISO date"`
	ConverterType   string         `json:"converterType" example:"javascript"`
	Source          map[string]any `json:"source"`
	IsSystem        bool           `json:"isSystem" example:"false"`
	Meta            map[string]any `json:"meta"`
	Active          bool           `json:"active" example:"true"`
	DeletedAt       *time.Time     `json:"deletedAt" example:"2026-07-23T10:00:00Z"`
	CreatedAt       time.Time      `json:"createdAt" example:"2026-07-23T10:00:00Z"`
	UpdatedAt       time.Time      `json:"updatedAt" example:"2026-07-23T10:00:00Z"`
}
type ConvertersListResponse struct {
	Items []*ConverterResponse `json:"items"`
}
