package i18n_bundle

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Locales map[string]any `json:"locales,omitempty"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Locales *map[string]any `json:"locales,omitempty"`
}

type Response struct {
	shared.DocumentMetadata
	Locales map[string]any `json:"locales"`
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
