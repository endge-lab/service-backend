package backup

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateRequest struct {
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000" example:"Перед обновлением production-конфигурации"`
}

type Response struct {
	ID            string         `json:"id" format:"uuid"`
	WorkspaceID   string         `json:"workspaceId" format:"uuid"`
	Kind          string         `json:"kind" enums:"manual,pre_import"`
	Description   *string        `json:"description,omitempty"`
	SchemaVersion int            `json:"schemaVersion" example:"1"`
	Checksum      string         `json:"checksum"`
	SizeBytes     int64          `json:"sizeBytes"`
	CreatedBy     entities.Actor `json:"createdBy"`
	CreatedAt     time.Time      `json:"createdAt" format:"date-time"`
	ExpiresAt     *time.Time     `json:"expiresAt,omitempty" format:"date-time"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total"`
}

type ExportResponse struct{ entities.PortableBundle }

func newResponse(value entities.SnapshotBackup) Response {
	return Response{ID: value.ID, WorkspaceID: value.WorkspaceID, Kind: value.Kind, Description: value.Description, SchemaVersion: value.SchemaVersion, Checksum: value.Checksum, SizeBytes: value.SizeBytes, CreatedBy: value.CreatedBy, CreatedAt: value.CreatedAt, ExpiresAt: value.ExpiresAt}
}
