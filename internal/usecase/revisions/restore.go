package revisions

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// Restore восстанавливает документ до выбранной ревизии.
func (s *UseCase) Restore(ctx context.Context, documentType, identity, revisionID string, expected int) (*entities.Document, error) {
	return s.coordinator.RestoreRevision(ctx, documentType, identity, revisionID, expected)
}
