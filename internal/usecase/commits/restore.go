package commits

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// PlanRestore строит предварительный план операции без изменения состояния.
func (s *UseCase) PlanRestore(ctx context.Context, id string) (*entities.ImportPlan, error) {
	return s.coordinator.PlanCommitRestore(ctx, id)
}

// Restore восстанавливает состояние рабочего пространства по коммиту.
func (s *UseCase) Restore(ctx context.Context, id string, expected int64) (*entities.Commit, error) {
	return s.coordinator.RestoreCommit(ctx, id, expected)
}
