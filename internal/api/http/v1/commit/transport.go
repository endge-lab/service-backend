package commit

import (
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	Message              string `json:"message" validate:"required,max=1000" example:"Сохранена конфигурация проекта"`
	RevisionPolicy       string `json:"revisionPolicy,omitempty" validate:"omitempty,oneof=preserve squash" example:"preserve" enums:"preserve,squash"`
	ExpectedHeadSequence *int64 `json:"expectedHeadSequence" validate:"required,gte=0" example:"42"`
}

type RestoreRequest struct {
	ExpectedHeadSequence *int64 `json:"expectedHeadSequence" validate:"required,gte=0" example:"42"`
}

type Response struct {
	ID             string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	WorkspaceID    string                  `json:"workspaceId" example:"550e8400-e29b-41d4-a716-446655440001" format:"uuid"`
	ParentCommitID *string                 `json:"parentCommitId,omitempty" example:"example" format:"uuid"`
	BaseSequence   int64                   `json:"baseSequence" example:"21"`
	HeadSequence   int64                   `json:"headSequence" example:"42"`
	Message        string                  `json:"message" example:"Сохранена конфигурация проекта"`
	RevisionPolicy string                  `json:"revisionPolicy" example:"preserve" enums:"preserve,squash"`
	Operation      string                  `json:"operation" example:"update"`
	CreatedBy      entities.Actor          `json:"createdBy"`
	CreatedAt      time.Time               `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	Changes        []entities.CommitChange `json:"changes,omitempty"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total" example:"1"`
}

type PlanResponse struct {
	BaseSequence  int64            `json:"baseSequence" example:"21"`
	HeadSequence  int64            `json:"headSequence" example:"42"`
	RevisionCount int              `json:"revisionCount" example:"3"`
	DocumentCount int              `json:"documentCount" example:"2"`
	Contributors  []entities.Actor `json:"contributors"`
	Shared        bool             `json:"shared" example:"false"`
}

type RestorePlanResponse struct {
	entities.ImportPlan
}

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Commit) (Response, error) {
	return shared.DecodeValue[Response](value)
}
