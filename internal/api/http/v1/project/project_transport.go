package project

import (
	"time"

	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Identity    string         `json:"identity" validate:"required,min=1,max=160" example:"example-project"`
	DisplayName string         `json:"displayName" validate:"required,min=1,max=255" example:"Example Project"`
	Description *string        `json:"description" example:"Project for local configuration"`
	Active      bool           `json:"active" example:"true"`
	Meta        map[string]any `json:"meta" swaggertype:"object"`
}

type UpdateProjectRequest struct {
	DisplayName string         `json:"displayName" validate:"required,min=1,max=255" example:"Demo Project"`
	Description *string        `json:"description,omitempty" example:"Updated description"`
	Active      bool           `json:"active" example:"true"`
	Meta        map[string]any `json:"meta" swaggertype:"object"`
}

type ProjectResponse struct {
	ID          uuid.UUID      `json:"id" example:"00000000-0000-4000-8000-000000000001"`
	Identity    string         `json:"identity" example:"demo-project"`
	DisplayName string         `json:"displayName" example:"Demo Project"`
	Description *string        `json:"description,omitempty" example:"Project for local configuration"`
	Active      bool           `json:"active" example:"true"`
	DeletedAt   *time.Time     `json:"deletedAt" example:"2026-07-08T10:00:00Z"`
	Meta        map[string]any `json:"meta" swaggertype:"object"`
	CreatedAt   time.Time      `json:"createdAt" example:"2026-07-08T10:00:00Z"`
	UpdatedAt   time.Time      `json:"updatedAt" example:"2026-07-08T10:00:00Z"`
}

type ProjectsListResponse struct {
	Items []*ProjectResponse `json:"items"`
}
