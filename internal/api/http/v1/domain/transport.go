package domain

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type ImportRequest struct {
	PlanID       string `json:"planId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440006" format:"uuid"`
	Confirmation string `json:"confirmation" validate:"required,max=160" example:"default"`
}

type ImportPlanRequest struct {
	Snapshot entities.PortableBundle `json:"snapshot" validate:"required"`
}

type ExportResponse struct {
	entities.PortableBundle
}

type ImportPlanResponse struct {
	entities.ImportPlan
}

type ImportResponse struct {
	Workspace       string                  `json:"workspace"`
	Backup          ImportBackupResponse    `json:"backup"`
	Imported        entities.SnapshotCounts `json:"imported"`
	InitialCommitID string                  `json:"initialCommitId" format:"uuid"`
}

// ImportBackupResponse содержит только metadata автоматического backup, без повторной
// отправки всего snapshot внутри ответа на импорт.
type ImportBackupResponse struct {
	ID            string         `json:"id" format:"uuid"`
	Kind          string         `json:"kind" enums:"pre_import"`
	Description   *string        `json:"description,omitempty"`
	SchemaVersion int            `json:"schemaVersion" example:"1"`
	Checksum      string         `json:"checksum"`
	SizeBytes     int64          `json:"sizeBytes"`
	CreatedBy     entities.Actor `json:"createdBy"`
	CreatedAt     time.Time      `json:"createdAt" format:"date-time"`
	ExpiresAt     *time.Time     `json:"expiresAt,omitempty" format:"date-time"`
}

func newImportResponse(value entities.SnapshotImportResult) ImportResponse {
	return ImportResponse{
		Workspace: value.WorkspaceIdentity,
		Backup: ImportBackupResponse{
			ID: value.Backup.ID, Kind: value.Backup.Kind, Description: value.Backup.Description,
			SchemaVersion: value.Backup.SchemaVersion, Checksum: value.Backup.Checksum,
			SizeBytes: value.Backup.SizeBytes, CreatedBy: value.Backup.CreatedBy,
			CreatedAt: value.Backup.CreatedAt, ExpiresAt: value.Backup.ExpiresAt,
		},
		Imported: value.Imported, InitialCommitID: value.InitialCommitID,
	}
}
