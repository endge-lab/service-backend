package folder

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type CreateFolderRequest struct {
	EntityType     entities.FolderEntityType `json:"entityType" validate:"required" enums:"components,converters,queries,data-views" example:"components"`
	Identity       string                    `json:"identity" validate:"required,min=1,max=160" example:"shared-components"`
	DisplayName    string                    `json:"displayName" validate:"required,min=1,max=255" example:"Shared Components"`
	Description    *string                   `json:"description,omitempty" example:"Reusable components"`
	ParentIdentity *string                   `json:"parentIdentity,omitempty" example:"root-components"`
	Meta           map[string]any            `json:"meta" swaggertype:"object"`
}

type UpdateFolderRequest struct {
	DisplayName    string         `json:"displayName" validate:"required,min=1,max=255" example:"Shared Components"`
	Description    *string        `json:"description,omitempty" example:"Reusable components"`
	ParentIdentity *string        `json:"parentIdentity,omitempty" example:"root-components"`
	Meta           map[string]any `json:"meta" swaggertype:"object"`
}

type FolderResponse struct {
	ID              uuid.UUID                 `json:"id" example:"00000000-0000-4000-8000-000000000011"`
	ProjectIdentity string                    `json:"projectIdentity" example:"demo-project"`
	EntityType      entities.FolderEntityType `json:"entityType" enums:"components,converters,queries,data-views" example:"components"`
	Identity        string                    `json:"identity" example:"shared-components"`
	DisplayName     string                    `json:"displayName" example:"Shared Components"`
	Description     *string                   `json:"description" example:"Reusable components"`
	ParentIdentity  *string                   `json:"parentIdentity" example:"root-components"`
	IsRoot          bool                      `json:"isRoot" example:"false"`
	IsSystem        bool                      `json:"isSystem" example:"false"`
	DeletedAt       *time.Time                `json:"deletedAt" example:"2026-07-08T10:00:00Z"`
	Meta            map[string]any            `json:"meta" swaggertype:"object"`
	CreatedAt       time.Time                 `json:"createdAt" example:"2026-07-08T10:00:00Z"`
	UpdatedAt       time.Time                 `json:"updatedAt" example:"2026-07-08T10:00:00Z"`
}

type FoldersListResponse struct {
	Items []*FolderResponse `json:"items"`
}
