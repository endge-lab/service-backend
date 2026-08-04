package workspace

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/workspaces"
)

type InstalledIntegration struct {
	Identity      string         `json:"identity" validate:"required,max=160" example:"main"`
	Version       string         `json:"version" validate:"required,max=160" example:"1.0.0"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

type CreateRequest struct {
	Identity              string                 `json:"identity" validate:"required,max=160" example:"main"`
	DisplayName           string                 `json:"displayName" validate:"required,max=255" example:"Основной объект"`
	Description           *string                `json:"description,omitempty" example:"Описание объекта"`
	DataMode              string                 `json:"dataMode,omitempty" validate:"omitempty,oneof=development production" example:"development" enums:"development,production"`
	Configuration         map[string]any         `json:"configuration,omitempty"`
	Meta                  map[string]any         `json:"meta,omitempty"`
	Active                *bool                  `json:"active,omitempty" example:"true"`
	InstalledIntegrations []InstalledIntegration `json:"installedIntegrations,omitempty" validate:"omitempty,dive"`
}

type PatchRequest struct {
	Identity              *string                 `json:"identity,omitempty" validate:"omitempty,max=160" example:"main"`
	DisplayName           *string                 `json:"displayName,omitempty" validate:"omitempty,max=255" example:"Основной объект"`
	Description           *string                 `json:"description,omitempty" example:"Описание объекта"`
	DataMode              *string                 `json:"dataMode,omitempty" validate:"omitempty,oneof=development production" example:"development" enums:"development,production"`
	Configuration         *map[string]any         `json:"configuration,omitempty"`
	Meta                  *map[string]any         `json:"meta,omitempty"`
	Active                *bool                   `json:"active,omitempty" example:"true"`
	InstalledIntegrations *[]InstalledIntegration `json:"installedIntegrations,omitempty" validate:"omitempty,dive"`
}

type MembershipRequest struct {
	Role string `json:"role" validate:"required,oneof=viewer editor admin" example:"admin" enums:"viewer,editor,admin"`
}

type Response struct {
	ID            string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	Identity      string          `json:"identity" example:"main"`
	DisplayName   string          `json:"displayName" example:"Основной объект"`
	Description   *string         `json:"description,omitempty" example:"Описание объекта"`
	DataMode      string          `json:"dataMode" example:"development" enums:"development,production"`
	Configuration json.RawMessage `json:"configuration" swaggertype:"object"`
	Meta          json.RawMessage `json:"meta" swaggertype:"object"`
	Active        bool            `json:"active" example:"true"`
	HeadSequence  int64           `json:"headSequence" example:"42"`
	Revision      int             `json:"revision" example:"3"`
	CreatedBy     entities.Actor  `json:"createdBy"`
	UpdatedBy     entities.Actor  `json:"updatedBy"`
	CreatedAt     time.Time       `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	UpdatedAt     time.Time       `json:"updatedAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
}

type MembershipResponse struct {
	WorkspaceID string    `json:"workspaceId" example:"550e8400-e29b-41d4-a716-446655440001" format:"uuid"`
	UserID      string    `json:"userId" example:"550e8400-e29b-41d4-a716-446655440002" format:"uuid"`
	Role        string    `json:"role" example:"admin" enums:"viewer,editor,admin"`
	Username    string    `json:"username,omitempty" example:"egor"`
	DisplayName string    `json:"displayName,omitempty" example:"Основной объект"`
	CreatedAt   time.Time `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total" example:"1"`
}

type MembershipListResponse struct {
	Items []MembershipResponse `json:"items"`
	Total int                  `json:"total" example:"1"`
}

// Input преобразует create DTO в application input.
func (r CreateRequest) Input() resourceusecase.CreateInput {
	return resourceusecase.CreateInput{
		Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description,
		DataMode: r.DataMode, Configuration: r.Configuration, Meta: r.Meta, Active: r.Active,
		InstalledIntegrations: integrationInputs(r.InstalledIntegrations),
	}
}

// Input преобразует проверенный raw PATCH в application input без потери explicit null.
func (r PatchRequest) Input(raw []byte) (resourceusecase.PatchInput, error) {
	return resourceusecase.NewPatchInputJSON(raw)
}

func integrationInputs(values []InstalledIntegration) []resourceusecase.InstalledIntegrationInput {
	result := make([]resourceusecase.InstalledIntegrationInput, 0, len(values))
	for _, value := range values {
		result = append(result, resourceusecase.InstalledIntegrationInput{
			Identity: value.Identity, Version: value.Version, Configuration: value.Configuration,
		})
	}
	return result
}

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Workspace) (Response, error) {
	return shared.DecodeValue[Response](value)
}

// NewMembershipResponse безопасно преобразует membership в HTTP-ответ.
func NewMembershipResponse(value entities.Membership) (MembershipResponse, error) {
	return shared.DecodeValue[MembershipResponse](value)
}
