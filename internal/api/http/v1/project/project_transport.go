package project

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Identity              string         `json:"identity" validate:"required,min=1,max=160" example:"swagger-project-new"`
	DisplayName           string         `json:"displayName" validate:"required,min=1,max=255" example:"Swagger Project New"`
	ExtendSettings        bool           `json:"extendSettings" example:"false"`
	SettingsID            *uuid.UUID     `json:"settingsId,omitempty" example:"00000000-0000-4000-8000-000000000003"`
	NavigationID          *uuid.UUID     `json:"navigationId,omitempty" example:"00000000-0000-4000-8000-000000000004"`
	FolderID              *uuid.UUID     `json:"folderId,omitempty" example:"00000000-0000-4000-8000-000000000002"`
	AllowedEnvironmentIDs []uuid.UUID    `json:"allowedEnvironmentIds" example:"00000000-0000-4000-8000-000000000005"`
	Meta                  map[string]any `json:"meta" swaggertype:"object"`
}

type UpdateProjectRequest struct {
	Identity              string         `json:"identity" validate:"required,min=1,max=160" example:"swagger-project"`
	DisplayName           string         `json:"displayName" validate:"required,min=1,max=255" example:"Swagger Project"`
	ExtendSettings        bool           `json:"extendSettings" example:"true"`
	SettingsID            *uuid.UUID     `json:"settingsId,omitempty" example:"00000000-0000-4000-8000-000000000003"`
	NavigationID          *uuid.UUID     `json:"navigationId,omitempty" example:"00000000-0000-4000-8000-000000000004"`
	FolderID              *uuid.UUID     `json:"folderId,omitempty" example:"00000000-0000-4000-8000-000000000002"`
	AllowedEnvironmentIDs []uuid.UUID    `json:"allowedEnvironmentIds" example:"00000000-0000-4000-8000-000000000005"`
	Meta                  map[string]any `json:"meta" swaggertype:"object"`
}

type ProjectResponse struct {
	ID                    uuid.UUID      `json:"id" example:"00000000-0000-4000-8000-000000000001"`
	Identity              string         `json:"identity" example:"swagger-project"`
	DisplayName           string         `json:"displayName" example:"Swagger Project"`
	ExtendSettings        bool           `json:"extendSettings" example:"false"`
	SettingsID            *uuid.UUID     `json:"settingsId,omitempty" example:"00000000-0000-4000-8000-000000000003"`
	NavigationID          *uuid.UUID     `json:"navigationId,omitempty" example:"00000000-0000-4000-8000-000000000004"`
	FolderID              *uuid.UUID     `json:"folderId,omitempty" example:"00000000-0000-4000-8000-000000000002"`
	AllowedEnvironmentIDs []uuid.UUID    `json:"allowedEnvironmentIds" example:"00000000-0000-4000-8000-000000000005"`
	DeletedAt             *time.Time     `json:"deletedAt,omitempty" example:"2026-07-06T14:53:00Z"`
	Meta                  map[string]any `json:"meta" swaggertype:"object"`
	CreatedAt             time.Time      `json:"createdAt" example:"2026-07-06T14:53:00Z"`
	UpdatedAt             time.Time      `json:"updatedAt" example:"2026-07-06T14:53:00Z"`
}

type ProjectsListResponse struct {
	Items []*ProjectResponse `json:"items"`
}

type CountProjectsResponse struct {
	Count int64 `json:"count" example:"42"`
}

func NewProjectResponse(project *entities.Project) *ProjectResponse {
	if project == nil {
		return nil
	}

	return &ProjectResponse{
		ID:                    project.ID,
		Identity:              project.Identity,
		DisplayName:           project.DisplayName,
		ExtendSettings:        project.ExtendSettings,
		SettingsID:            project.SettingsID,
		NavigationID:          project.NavigationID,
		FolderID:              project.FolderID,
		AllowedEnvironmentIDs: project.AllowedEnvironmentIDs,
		DeletedAt:             project.DeletedAt,
		Meta:                  project.Meta,
		CreatedAt:             project.CreatedAt,
		UpdatedAt:             project.UpdatedAt,
	}
}

func NewProjectsListResponse(projects []*entities.Project) ProjectsListResponse {
	items := make([]*ProjectResponse, 0, len(projects))
	for _, item := range projects {
		items = append(items, NewProjectResponse(item))
	}

	return ProjectsListResponse{Items: items}
}
