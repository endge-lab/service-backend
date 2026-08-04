package session

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/session"
)

type UserResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	ProviderID  string    `json:"providerId" example:"keycloak"`
	Subject     string    `json:"subject" example:"user-subject"`
	Issuer      string    `json:"issuer" example:"https://id.example.com/realms/endge"`
	Username    string    `json:"username,omitempty" example:"egor"`
	DisplayName string    `json:"displayName,omitempty" example:"Основной объект"`
	Role        string    `json:"role,omitempty" example:"admin" enums:"viewer,editor,admin"`
	Active      bool      `json:"active" example:"true"`
	LastSeenAt  time.Time `json:"lastSeenAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
	CreatedAt   time.Time `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
}

type WorkspaceResponse struct {
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

type Response struct {
	User          *UserResponse       `json:"user"`
	PlatformAdmin bool                `json:"platformAdmin" example:"true"`
	Workspaces    []WorkspaceResponse `json:"workspaces"`
}

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value resourceusecase.Result) (Response, error) {
	return shared.DecodeValue[Response](value)
}
