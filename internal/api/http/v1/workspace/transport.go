package http

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"time"
)

type CreateRequest struct {
	Identity      string                       `json:"identity" validate:"required,min=1,max=160"`
	DisplayName   string                       `json:"displayName" validate:"required,min=1,max=255"`
	Configuration *entities.EndgeConfiguration `json:"configuration"`
}
type UpdateRequest struct {
	DisplayName   *string                      `json:"displayName"`
	Configuration *entities.EndgeConfiguration `json:"configuration"`
}
type Response struct {
	ID            uuid.UUID                   `json:"id"`
	Identity      string                      `json:"identity"`
	DisplayName   string                      `json:"displayName"`
	Configuration entities.EndgeConfiguration `json:"configuration"`
	CreatedAt     time.Time                   `json:"createdAt"`
	UpdatedAt     time.Time                   `json:"updatedAt"`
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
