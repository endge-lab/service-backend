package domain

import "github.com/endge-lab/service-backend/internal/domain/entities"

type UsageResponse struct {
	OwnerType         string `json:"ownerType" example:"type"`
	OwnerID           string `json:"ownerId" example:"550e8400-e29b-41d4-a716-446655440000"`
	OwnerIdentity     string `json:"ownerIdentity" example:"OrderList"`
	SourcePath        string `json:"sourcePath" example:"schema.fields[0].type"`
	VerificationState string `json:"verificationState" example:"verified"`
}

type UsagesListResponse struct {
	Items  []UsageResponse `json:"items"`
	Total  int64           `json:"total" example:"1"`
	Limit  int             `json:"limit" example:"50"`
	Offset int             `json:"offset" example:"0"`
}

func usageResponse(value entities.DomainDependencyUsage) UsageResponse {
	return UsageResponse{OwnerType: value.OwnerType, OwnerID: value.OwnerID.String(), OwnerIdentity: value.OwnerIdentity, SourcePath: value.SourcePath, VerificationState: string(value.VerificationState)}
}

func usagesListResponse(value entities.DomainDependencyUsages) UsagesListResponse {
	items := make([]UsageResponse, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, usageResponse(item))
	}
	return UsagesListResponse{Items: items, Total: value.Total, Limit: value.Limit, Offset: value.Offset}
}
