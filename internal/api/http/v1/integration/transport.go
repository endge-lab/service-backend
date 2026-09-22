package integration

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/integrations"
)

type CreateRequest struct {
	Identity    string         `json:"identity" validate:"required,max=160" example:"main"`
	DisplayName string         `json:"displayName" validate:"required,max=255" example:"Основной объект"`
	Description *string        `json:"description,omitempty" example:"Описание объекта"`
	Version     string         `json:"version" validate:"required,max=160" example:"1.0.0"`
	ManagedBy   string         `json:"managedBy,omitempty" validate:"omitempty,oneof=user system integration" example:"user" enums:"user,system,integration"`
	ManagedByID *string        `json:"managedById,omitempty" example:"endge-core"`
	Meta        map[string]any `json:"meta,omitempty"`
	Active      *bool          `json:"active,omitempty" example:"true"`
}

type PatchRequest struct {
	Identity    *string         `json:"identity,omitempty" validate:"omitempty,max=160" example:"main"`
	DisplayName *string         `json:"displayName,omitempty" validate:"omitempty,max=255" example:"Основной объект"`
	Description *string         `json:"description,omitempty" example:"Описание объекта"`
	Version     *string         `json:"version,omitempty" validate:"omitempty,max=160" example:"1.0.0"`
	ManagedBy   *string         `json:"managedBy,omitempty" validate:"omitempty,oneof=user system integration" example:"user" enums:"user,system,integration"`
	ManagedByID *string         `json:"managedById,omitempty" example:"endge-core"`
	Meta        *map[string]any `json:"meta,omitempty"`
	Active      *bool           `json:"active,omitempty" example:"true"`
}

type Response struct {
	ID          string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	Identity    string          `json:"identity" example:"main"`
	DisplayName string          `json:"displayName" example:"Основной объект"`
	Description *string         `json:"description,omitempty" example:"Описание объекта"`
	Version     string          `json:"version" example:"1.0.0"`
	ManagedBy   string          `json:"managedBy" example:"user" enums:"user,system,integration"`
	ManagedByID *string         `json:"managedById,omitempty" example:"endge-core"`
	Meta        json.RawMessage `json:"meta" swaggertype:"object"`
	Active      bool            `json:"active" example:"true"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty" example:"2026-08-04T11:00:00Z" format:"date-time"`
	Revision    int             `json:"revision" example:"3"`
	CreatedBy   entities.Actor  `json:"createdBy"`
	UpdatedBy   entities.Actor  `json:"updatedBy"`
	CreatedAt   time.Time       `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	UpdatedAt   time.Time       `json:"updatedAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total" example:"1"`
}

// Input преобразует create DTO в application input.
func (r CreateRequest) Input() resourceusecase.CreateInput {
	return resourceusecase.CreateInput{
		Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description,
		Version: r.Version, ManagedBy: r.ManagedBy, ManagedByID: r.ManagedByID,
		Meta: r.Meta, Active: r.Active,
	}
}

// Input преобразует проверенный raw PATCH в application input без потери explicit null.
func (r PatchRequest) Input(raw []byte) (resourceusecase.PatchInput, error) {
	return resourceusecase.NewPatchInputJSON(raw)
}

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Integration) (Response, error) {
	return shared.DecodeValue[Response](value)
}
