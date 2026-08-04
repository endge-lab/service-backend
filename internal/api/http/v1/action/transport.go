package action

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Definition            json.RawMessage `json:"definition,omitempty" swaggertype:"object"`
	Input                 json.RawMessage `json:"input,omitempty" swaggertype:"object"`
	Output                json.RawMessage `json:"output,omitempty" swaggertype:"object"`
	Target                json.RawMessage `json:"target,omitempty" swaggertype:"object"`
	DefaultImplementation json.RawMessage `json:"defaultImplementation,omitempty" swaggertype:"object"`
	Owner                 json.RawMessage `json:"owner,omitempty" swaggertype:"object"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Definition            json.RawMessage `json:"definition,omitempty" swaggertype:"object"`
	Input                 json.RawMessage `json:"input,omitempty" swaggertype:"object"`
	Output                json.RawMessage `json:"output,omitempty" swaggertype:"object"`
	Target                json.RawMessage `json:"target,omitempty" swaggertype:"object"`
	DefaultImplementation json.RawMessage `json:"defaultImplementation,omitempty" swaggertype:"object"`
	Owner                 json.RawMessage `json:"owner,omitempty" swaggertype:"object"`
}

type Response struct {
	shared.DocumentMetadata
	Definition            json.RawMessage `json:"definition" swaggertype:"object"`
	Input                 json.RawMessage `json:"input" swaggertype:"object"`
	Output                json.RawMessage `json:"output" swaggertype:"object"`
	Target                json.RawMessage `json:"target" swaggertype:"object"`
	DefaultImplementation json.RawMessage `json:"defaultImplementation" swaggertype:"object"`
	Owner                 json.RawMessage `json:"owner" swaggertype:"object"`
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
