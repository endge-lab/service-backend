package auth_profile

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	AdapterID   string            `json:"adapterId" validate:"required,max=160" example:"oidc"`
	Config      map[string]any    `json:"config,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
	Session     *SessionPolicy    `json:"session,omitempty"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	AdapterID   *string            `json:"adapterId,omitempty" validate:"omitempty,max=160" example:"oidc"`
	Config      *map[string]any    `json:"config,omitempty"`
	Credentials *map[string]string `json:"credentials,omitempty"`
	Session     *SessionPolicy     `json:"session,omitempty"`
}

type Response struct {
	shared.DocumentMetadata
	AdapterID   string            `json:"adapterId" example:"oidc"`
	Config      map[string]any    `json:"config"`
	Credentials map[string]string `json:"credentials"`
	Session     *SessionPolicy    `json:"session,omitempty"`
}

// SessionPolicy задаёт browser storage для token-producing адаптеров.
type SessionPolicy struct {
	Storage             string `json:"storage" validate:"required,oneof=localStorage sessionStorage memory" example:"memory"`
	PersistRefreshToken bool   `json:"persistRefreshToken" example:"false"`
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
