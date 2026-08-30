package domain

import "github.com/endge-lab/service-backend/internal/domain/entities"

type ImportRequest struct {
	PlanID       string `json:"planId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440006" format:"uuid"`
	Confirmation string `json:"confirmation" validate:"required,max=160" example:"default"`
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
	DomainVersion  string                  `json:"domainVersion" example:"dv2:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`
}

func newImportResponse(value entities.SnapshotImportResult) ImportResponse {
	return ImportResponse{
		Workspace: value.WorkspaceIdentity, Imported: value.Imported,
		Creates: value.Creates, Updates: value.Updates, Restores: value.Restores, Deletes: value.Deletes,
		CommitID: value.CommitID, ParentCommitID: value.ParentCommitID, DomainVersion: value.DomainVersion,
	}
}
