package domain_type

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	IsPrimitive   bool   `json:"isPrimitive,omitempty" example:"true"`
	Source        string `json:"source" validate:"required,max=8388608" example:"export default {}"`
	SourceVersion int    `json:"sourceVersion" validate:"required,gt=0" example:"2"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	IsPrimitive   *bool   `json:"isPrimitive,omitempty" example:"true"`
	Source        *string `json:"source,omitempty" validate:"omitempty,max=8388608" example:"export default {}"`
	SourceVersion *int    `json:"sourceVersion,omitempty" validate:"omitempty,gt=0" example:"2"`
}

type Response struct {
	shared.DocumentMetadata
	IsPrimitive   bool   `json:"isPrimitive" example:"true"`
	Source        string `json:"source" example:"export default {}"`
	SourceVersion int    `json:"sourceVersion" example:"2"`
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
