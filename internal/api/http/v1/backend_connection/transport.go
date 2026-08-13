package backend_connection

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	Name    string `json:"name" validate:"required,max=160" example:"Production"`
	BaseURL string `json:"baseUrl" validate:"required" example:"https://backend.example.com"`
}

type Response struct {
	ID        string    `json:"id" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name" example:"Production"`
	BaseURL   string    `json:"baseUrl" example:"https://backend.example.com"`
	CreatedBy string    `json:"createdBy" format:"uuid"`
	CreatedAt time.Time `json:"createdAt" format:"date-time"`
}

type ListResponse struct {
	Items     []Response `json:"items"`
	Total     int        `json:"total" example:"1"`
	CanManage bool       `json:"canManage" example:"true"`
}

func newResponse(value entities.BackendConnection) Response {
	return Response(value)
}
