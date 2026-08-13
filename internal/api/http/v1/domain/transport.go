package domain

import "github.com/endge-lab/service-backend/internal/domain/entities"

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
	Workspace      string                  `json:"workspace"`
	Imported       entities.SnapshotCounts `json:"imported"`
	Creates        int                     `json:"creates"`
	Updates        int                     `json:"updates"`
	Restores       int                     `json:"restores"`
	Deletes        int                     `json:"deletes"`
	CommitID       string                  `json:"commitId" format:"uuid"`
	ParentCommitID string                  `json:"parentCommitId" format:"uuid"`
}

func newImportResponse(value entities.SnapshotImportResult) ImportResponse {
	return ImportResponse{
		Workspace: value.WorkspaceIdentity, Imported: value.Imported,
		Creates: value.Creates, Updates: value.Updates, Restores: value.Restores, Deletes: value.Deletes,
		CommitID: value.CommitID, ParentCommitID: value.ParentCommitID,
	}
}
