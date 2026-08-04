package revision

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type Response struct {
	ID                     string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	WorkspaceID            string           `json:"workspaceId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001" format:"uuid"`
	DocumentType           string           `json:"documentType" example:"example"`
	DocumentID             string           `json:"documentId" example:"550e8400-e29b-41d4-a716-446655440003" format:"uuid"`
	DocumentIdentity       string           `json:"documentIdentity" example:"example"`
	RevisionNumber         int              `json:"revisionNumber" example:"3"`
	WorkspaceSequence      *int64           `json:"workspaceSequence,omitempty" example:"1"`
	Operation              string           `json:"operation" example:"update"`
	ParentRevisionID       *string          `json:"parentRevisionId,omitempty" example:"example" format:"uuid"`
	RestoredFromRevisionID *string          `json:"restoredFromRevisionId,omitempty" example:"example" format:"uuid"`
	CommittedInCommitID    *string          `json:"committedInCommitId,omitempty" example:"example" format:"uuid"`
	MutationBatchID        string           `json:"mutationBatchId" example:"550e8400-e29b-41d4-a716-446655440005" format:"uuid"`
	SnapshotVersion        int              `json:"snapshotVersion" example:"1"`
	Snapshot               json.RawMessage  `json:"snapshot" swaggertype:"object"`
	Checksum               string           `json:"checksum" example:"sha256:0123456789abcdef"`
	CreatedBy              entities.Actor   `json:"createdBy"`
	CreatedAt              time.Time        `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	Contributors           []entities.Actor `json:"contributors,omitempty"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	Total int        `json:"total" example:"1"`
}

type RestoreResponse map[string]any

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Revision) (Response, error) {
	return shared.DecodeValue[Response](value)
}
