package auth_profile

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	AdapterID      string            `json:"adapterId" validate:"required,max=160" example:"example"`
	Config         map[string]any    `json:"config,omitempty"`
	CredentialRefs map[string]string `json:"credentialRefs,omitempty"`
	Persist        string            `json:"persist" validate:"required,oneof=localStorage sessionStorage memory" example:"example"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	AdapterID      *string            `json:"adapterId,omitempty" validate:"omitempty,max=160" example:"example"`
	Config         *map[string]any    `json:"config,omitempty"`
	CredentialRefs *map[string]string `json:"credentialRefs,omitempty"`
	Persist        *string            `json:"persist,omitempty" validate:"omitempty,oneof=localStorage sessionStorage memory" example:"example"`
}

type Response struct {
	shared.DocumentMetadata
	AdapterID      string            `json:"adapterId" example:"example"`
	Config         map[string]any    `json:"config"`
	CredentialRefs map[string]string `json:"credentialRefs"`
	Persist        string            `json:"persist" example:"example"`
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
