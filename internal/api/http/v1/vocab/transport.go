package vocab

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Mode                string  `json:"mode,omitempty" validate:"omitempty,oneof=external_payload internal" example:"example"`
	BaseAPIURL          *string `json:"baseApiUrl,omitempty" example:"example"`
	CollectionSlug      *string `json:"collectionSlug,omitempty" example:"example"`
	AuthMode            string  `json:"authMode,omitempty" validate:"omitempty,oneof=inherit profile manual none" example:"example"`
	AuthProfileIdentity *string `json:"authProfileIdentity,omitempty" example:"default-auth"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Mode                *string `json:"mode,omitempty" validate:"omitempty,oneof=external_payload internal" example:"example"`
	BaseAPIURL          *string `json:"baseApiUrl,omitempty" example:"example"`
	CollectionSlug      *string `json:"collectionSlug,omitempty" example:"example"`
	AuthMode            *string `json:"authMode,omitempty" validate:"omitempty,oneof=inherit profile manual none" example:"example"`
	AuthProfileIdentity *string `json:"authProfileIdentity,omitempty" example:"default-auth"`
}

type Response struct {
	shared.DocumentMetadata
	Mode                string  `json:"mode" example:"example"`
	BaseAPIURL          *string `json:"baseApiUrl,omitempty" example:"example"`
	CollectionSlug      *string `json:"collectionSlug,omitempty" example:"example"`
	AuthMode            string  `json:"authMode" example:"example"`
	AuthProfileIdentity *string `json:"authProfileIdentity,omitempty" example:"default-auth"`
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
