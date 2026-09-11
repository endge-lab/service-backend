package folder

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	shared.CreateDocumentRequest
	Scope          string  `json:"scope,omitempty" validate:"omitempty,oneof=collection workspace" example:"collection" enums:"collection,workspace"`
	EntityType     *string `json:"entityType,omitempty" example:"projects"`
	ParentIdentity *string `json:"parentIdentity,omitempty" example:"root-projects"`
	Icon           *string `json:"icon,omitempty" validate:"omitempty,max=80" example:"FolderKanban"`
	Color          *string `json:"color,omitempty" validate:"omitempty,len=7" example:"#64748b"`
}

type PatchRequest struct {
	shared.PatchDocumentRequest
	Scope          *string `json:"scope,omitempty" validate:"omitempty,oneof=collection workspace" enums:"collection,workspace"`
	EntityType     *string `json:"entityType,omitempty" example:"projects"`
	ParentIdentity *string `json:"parentIdentity,omitempty" example:"root-projects"`
	Icon           *string `json:"icon,omitempty" validate:"omitempty,max=80"`
	Color          *string `json:"color,omitempty" validate:"omitempty,len=7"`
}

type Response struct {
	shared.DocumentMetadata
	Scope          string  `json:"scope" enums:"collection,workspace"`
	EntityType     *string `json:"entityType,omitempty" example:"projects"`
	ParentIdentity *string `json:"parentIdentity,omitempty" example:"root-projects"`
	Icon           *string `json:"icon,omitempty"`
	Color          *string `json:"color,omitempty"`
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
