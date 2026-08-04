package release

import (
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/releases"
)

type CreateRequest struct {
	Identity       string  `json:"identity" validate:"required,max=160" example:"main"`
	DisplayName    string  `json:"displayName" validate:"required,max=255" example:"Основной объект"`
	Description    *string `json:"description,omitempty" example:"Описание объекта"`
	SourceCommitID string  `json:"sourceCommitId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440006" format:"uuid"`
}

type RestoreRequest struct {
	ExpectedHeadSequence *int64 `json:"expectedHeadSequence" validate:"required,gte=0" example:"42"`
}

type Response struct {
	ID             string         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	WorkspaceID    string         `json:"workspaceId" example:"550e8400-e29b-41d4-a716-446655440001" format:"uuid"`
	Identity       string         `json:"identity" example:"main"`
	DisplayName    string         `json:"displayName" example:"Основной объект"`
	Description    *string        `json:"description,omitempty" example:"Описание объекта"`
	SourceCommitID string         `json:"sourceCommitId" example:"550e8400-e29b-41d4-a716-446655440006" format:"uuid"`
	HeadSequence   int64          `json:"headSequence" example:"42"`
	SchemaVersion  int            `json:"schemaVersion" example:"1"`
	Checksum       string         `json:"checksum" example:"sha256:0123456789abcdef"`
	CreatedBy      entities.Actor `json:"createdBy"`
	CreatedAt      time.Time      `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total" example:"1"`
}

type RestorePlanResponse struct {
	entities.ImportPlan
}

type RestoreResponse struct {
	entities.Commit
}

type ExportResponse struct {
	entities.PortableBundle
}

// Input преобразует release DTO в application input.
func (r CreateRequest) Input() resourceusecase.CreateInput {
	return resourceusecase.CreateInput{
		Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description, SourceCommitID: r.SourceCommitID,
	}
}

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Release) (Response, error) {
	return shared.DecodeValue[Response](value)
}
