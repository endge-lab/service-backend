package facet

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/facets"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type CreateRequest struct {
	Identity    string         `json:"identity" validate:"required,max=160" example:"region"`
	DisplayName string         `json:"displayName" validate:"required,max=255" example:"Регион"`
	Icon        string         `json:"icon" validate:"required,max=80" example:"MapPin"`
	Color       string         `json:"color" validate:"required,len=7" example:"#2563eb"`
	Meta        map[string]any `json:"meta,omitempty"`
}

type PatchRequest struct {
	Identity    *string         `json:"identity,omitempty" validate:"omitempty,max=160"`
	DisplayName *string         `json:"displayName,omitempty" validate:"omitempty,max=255"`
	Icon        *string         `json:"icon,omitempty" validate:"omitempty,max=80"`
	Color       *string         `json:"color,omitempty" validate:"omitempty,len=7"`
	Meta        *map[string]any `json:"meta,omitempty"`
}

type ReorderItem struct {
	Identity         string `json:"identity" validate:"required,max=160"`
	ExpectedRevision int    `json:"expectedRevision" validate:"required,min=1"`
}

type ReorderRequest struct {
	Items []ReorderItem `json:"items" validate:"required,max=1000"`
}

type DocumentCreateRequest struct {
	Identity      string         `json:"identity" validate:"required,max=160"`
	DisplayName   string         `json:"displayName" validate:"required,max=255"`
	Description   *string        `json:"description,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
	Active        *bool          `json:"active,omitempty"`
}

type DocumentPatchRequest struct {
	Identity      *string         `json:"identity,omitempty" validate:"omitempty,max=160"`
	DisplayName   *string         `json:"displayName,omitempty" validate:"omitempty,max=255"`
	Description   *string         `json:"description,omitempty"`
	Configuration *map[string]any `json:"configuration,omitempty"`
	Meta          *map[string]any `json:"meta,omitempty"`
	Active        *bool           `json:"active,omitempty"`
}

type FacetResponse struct {
	ID            string          `json:"id" format:"uuid"`
	Identity      string          `json:"identity"`
	DisplayName   string          `json:"displayName"`
	Icon          string          `json:"icon"`
	Color         string          `json:"color"`
	Position      int             `json:"position"`
	DocumentCount int             `json:"documentCount"`
	Meta          json.RawMessage `json:"meta" swaggertype:"object"`
	Active        bool            `json:"active"`
	DeletedAt     *time.Time      `json:"deletedAt,omitempty" format:"date-time"`
	Revision      int             `json:"revision"`
	CreatedBy     entities.Actor  `json:"createdBy"`
	UpdatedBy     entities.Actor  `json:"updatedBy"`
	CreatedAt     time.Time       `json:"createdAt" format:"date-time"`
	UpdatedAt     time.Time       `json:"updatedAt" format:"date-time"`
}

type FacetDocumentResponse struct {
	shared.DocumentMetadata
	FacetIdentity string         `json:"facetIdentity"`
	Configuration map[string]any `json:"configuration"`
}

type FacetListResponse struct {
	Items []FacetResponse `json:"items"`
	Total int             `json:"total"`
}

type FacetDocumentListResponse struct {
	Items  []FacetDocumentResponse `json:"items"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func NewFacetResponse(document entities.Document) (FacetResponse, error) {
	return shared.DecodeDocument[FacetResponse](document)
}

func NewFacetDocumentResponse(document entities.Document) (FacetDocumentResponse, error) {
	return shared.DecodeDocument[FacetDocumentResponse](document)
}

func (r CreateRequest) Input() resourceusecase.FacetCreateInput {
	return resourceusecase.FacetCreateInput{Identity: r.Identity, DisplayName: r.DisplayName, Icon: r.Icon, Color: r.Color, Meta: r.Meta}
}

func (r PatchRequest) Input() resourceusecase.FacetPatchInput {
	return resourceusecase.FacetPatchInput{Identity: r.Identity, DisplayName: r.DisplayName, Icon: r.Icon, Color: r.Color, Meta: r.Meta}
}

func (r ReorderRequest) Input() []ports.FacetOrderItem {
	result := make([]ports.FacetOrderItem, 0, len(r.Items))
	for _, item := range r.Items {
		result = append(result, ports.FacetOrderItem{Identity: item.Identity, ExpectedRevision: item.ExpectedRevision})
	}
	return result
}

func (r DocumentCreateRequest) Input() resourceusecase.FacetDocumentCreateInput {
	return resourceusecase.FacetDocumentCreateInput{Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description, Configuration: r.Configuration, Meta: r.Meta, Active: r.Active}
}

func (r DocumentPatchRequest) Input() resourceusecase.FacetDocumentPatchInput {
	return resourceusecase.FacetDocumentPatchInput{Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description, Configuration: r.Configuration, Meta: r.Meta, Active: r.Active}
}
