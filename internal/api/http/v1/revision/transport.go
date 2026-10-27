package revision

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

type Response struct {
	ID                     string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	WorkspaceID            string           `json:"workspaceId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001" format:"uuid"`
	DocumentType           string           `json:"documentType" example:"types"`
	DocumentID             string           `json:"documentId" example:"550e8400-e29b-41d4-a716-446655440003" format:"uuid"`
	DocumentIdentity       string           `json:"documentIdentity" example:"main"`
	RevisionNumber         int              `json:"revisionNumber" example:"3"`
	WorkspaceSequence      *int64           `json:"workspaceSequence,omitempty" example:"1"`
	Operation              string           `json:"operation" example:"update"`
	ParentRevisionID       *string          `json:"parentRevisionId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	RestoredFromRevisionID *string          `json:"restoredFromRevisionId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	CommittedInCommitID    *string          `json:"committedInCommitId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	MutationBatchID        string           `json:"mutationBatchId" example:"550e8400-e29b-41d4-a716-446655440005" format:"uuid"`
	SnapshotVersion        int              `json:"snapshotVersion" example:"1"`
	Snapshot               json.RawMessage  `json:"snapshot" swaggertype:"object"`
	Checksum               string           `json:"checksum" example:"sha256:0123456789abcdef"`
	CreatedBy              entities.Actor   `json:"createdBy"`
	AccessTokenID          *string          `json:"accessTokenId,omitempty" example:"550e8400-e29b-41d4-a716-446655440006" format:"uuid"`
	AccessTokenName        *string          `json:"accessTokenName,omitempty" example:"Codex: правка запросов"`
	CreatedAt              time.Time        `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	Contributors           []entities.Actor `json:"contributors,omitempty"`
}

type ListResponse struct {
	Items []Response `json:"items"`
	// Total — общее число ревизий документа, а не размер страницы.
	Total  int `json:"total" example:"1"`
	Limit  int `json:"limit,omitempty" example:"20"`
	Offset int `json:"offset" example:"0"`
}

// parseListOptions читает snapshot, limit и offset истории ревизий.
// Без limit сохраняется прежний контракт: вся история со снимками.
func parseListOptions(c *fiber.Ctx) (ports.RevisionListOptions, error) {
	options := ports.RevisionListOptions{IncludeSnapshot: true}
	if raw := strings.TrimSpace(c.Query("snapshot")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return options, domainerrors.InvalidInput("snapshot_invalid", "snapshot must be boolean")
		}
		options.IncludeSnapshot = value
	}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 500 {
			return options, domainerrors.InvalidInput("limit_invalid", "limit must be between 1 and 500")
		}
		options.Limit = value
	}
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return options, domainerrors.InvalidInput("offset_invalid", "offset must be non-negative")
		}
		options.Offset = value
	}
	return options, nil
}

type RestoreResponse map[string]any

// NewResponse безопасно преобразует application-результат в HTTP-ответ.
func NewResponse(value entities.Revision) (Response, error) {
	return shared.DecodeValue[Response](value)
}
