package releases

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// PlanRestore строит предварительный план операции без изменения состояния.
func (s *UseCase) PlanRestore(ctx context.Context, identity string) (*entities.ImportPlan, error) {
	if identity == "last" {
		return nil, domainerrors.InvalidInput("release_last_read_only", "last alias is available only for read operations")
	}
	return s.coordinator.PlanReleaseRestore(ctx, identity)
}

// Restore восстанавливает состояние рабочего пространства по релизу.
func (s *UseCase) Restore(ctx context.Context, identity string, expected int64) (*entities.Commit, error) {
	if identity == "last" {
		return nil, domainerrors.InvalidInput("release_last_read_only", "last alias is available only for read operations")
	}
	return s.coordinator.RestoreRelease(ctx, identity, expected)
}
