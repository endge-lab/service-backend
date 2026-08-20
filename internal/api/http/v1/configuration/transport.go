package configuration

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Source        string `json:"source" validate:"required,max=8388608" example:"defineConfig({})"`
	SourceVersion int    `json:"sourceVersion" validate:"required,eq=1" example:"1"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Source        *string `json:"source,omitempty" validate:"omitempty,max=8388608" example:"defineConfig({})"`
	SourceVersion *int    `json:"sourceVersion,omitempty" validate:"omitempty,eq=1" example:"1"`
}

type Response struct {
	shared.DocumentMetadata
	Source        string `json:"source" example:"defineConfig({})"`
	SourceVersion int    `json:"sourceVersion" example:"1"`
}

type ListResponse struct {
	Items  []Response `json:"items"`
	Total  int        `json:"total" example:"1"`
	Limit  int        `json:"limit" example:"100"`
	Offset int        `json:"offset" example:"0"`
}

func NewResponse(document entities.Document) (Response, error) {
	return shared.DecodeDocument[Response](document)
}
