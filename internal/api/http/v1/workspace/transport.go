package http

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"time"
)

type CreateRequest struct {
	Identity      string                       `json:"identity" validate:"required,min=1,max=160" example:"example-workspace"`
	DisplayName   string                       `json:"displayName" validate:"required,min=1,max=255" example:"Example Workspace"`
	Configuration *entities.EndgeConfiguration `json:"configuration"`
}
type UpdateRequest struct {
	DisplayName   *string                      `json:"displayName" example:"Updated Demo Workspace"`
	Configuration *entities.EndgeConfiguration `json:"configuration"`
}
type Response struct {
	ID            uuid.UUID                   `json:"id" example:"00000000-0000-4000-8000-000000000001"`
	Identity      string                      `json:"identity" example:"demo-workspace"`
	DisplayName   string                      `json:"displayName" example:"Demo Workspace"`
	Configuration entities.EndgeConfiguration `json:"configuration"`
	CreatedAt     time.Time                   `json:"createdAt" example:"2026-07-23T10:00:00Z"`
	UpdatedAt     time.Time                   `json:"updatedAt" example:"2026-07-23T10:00:00Z"`
}

func response(v *entities.RWorkspace) *Response {
	if v == nil {
		return nil
	}
	c := v.Configuration
	if c.SSE != nil {
		s := *c.SSE
		s.ManualToken = nil
		c.SSE = &s
	}
	return &Response{v.ID, v.Identity, v.DisplayName, c, v.CreatedAt, v.UpdatedAt}
}
