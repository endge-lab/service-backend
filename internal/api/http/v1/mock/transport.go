package mock

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	ContentSource string  `json:"contentSource,omitempty" example:"inline"`
	ContentType   string  `json:"contentType,omitempty" example:"application/json"`
	Source        string  `json:"source,omitempty" validate:"omitempty,max=8388608" example:"export default {}"`
	CodeRef       *string `json:"codeRef,omitempty" example:"fixtures/example.json"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	ContentSource *string `json:"contentSource,omitempty" example:"inline"`
	ContentType   *string `json:"contentType,omitempty" example:"application/json"`
	Source        *string `json:"source,omitempty" validate:"omitempty,max=8388608" example:"export default {}"`
	CodeRef       *string `json:"codeRef,omitempty" example:"fixtures/example.json"`
}

type Response struct {
	shared.DocumentMetadata
	ContentSource string  `json:"contentSource" example:"inline"`
	ContentType   string  `json:"contentType" example:"application/json"`
	Source        string  `json:"source" example:"export default {}"`
	CodeRef       *string `json:"codeRef,omitempty" example:"fixtures/example.json"`
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
