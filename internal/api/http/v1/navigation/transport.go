package navigation

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Tree []json.RawMessage `json:"tree,omitempty" swaggertype:"array,object"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Tree *[]json.RawMessage `json:"tree,omitempty" swaggertype:"array,object"`
}

type Response struct {
	shared.DocumentMetadata
	Tree []json.RawMessage `json:"tree" swaggertype:"array,object"`
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
