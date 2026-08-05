package project

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Configuration       map[string]any `json:"configuration,omitempty"`
	Slug                *string        `json:"slug,omitempty" validate:"omitempty,max=160"`
	Order               *int           `json:"order,omitempty"`
	NavigationIdentity  *string        `json:"navigationIdentity,omitempty" validate:"omitempty,max=160"`
	AllowedEnvironments []string       `json:"allowedEnvironments,omitempty" validate:"omitempty,dive,min=1,max=160" example:"development,production"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Configuration       *map[string]any `json:"configuration,omitempty"`
	Slug                *string         `json:"slug,omitempty" validate:"omitempty,max=160"`
	Order               *int            `json:"order,omitempty"`
	NavigationIdentity  *string         `json:"navigationIdentity,omitempty" validate:"omitempty,max=160"`
	AllowedEnvironments *[]string       `json:"allowedEnvironments,omitempty" validate:"omitempty,dive,min=1,max=160"`
}

type Response struct {
	shared.DocumentMetadata
	Configuration       map[string]any `json:"configuration"`
	Slug                *string        `json:"slug,omitempty"`
	Order               *int           `json:"order,omitempty"`
	NavigationIdentity  *string        `json:"navigationIdentity,omitempty"`
	AllowedEnvironments []string       `json:"allowedEnvironments" example:"development,production"`
}

type ListResponse struct {
	Items  []Response `json:"items"`
	Total  int        `json:"total" example:"1"`
	Limit  int        `json:"limit" example:"100"`
	Offset int        `json:"offset" example:"0"`
}

// NewResponse безопасно преобразует доменный документ в HTTP-ответ ресурса.
func NewResponse(document entities.Document) (Response, error) {
	return shared.DecodeDocument[Response](document)
}
