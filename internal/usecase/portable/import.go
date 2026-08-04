package portable

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// PlanImport строит предварительный план операции без изменения состояния.
func (s *UseCase) PlanImport(ctx context.Context, bundle entities.PortableBundle) (*entities.ImportPlan, error) {
	return s.coordinator.PlanImport(ctx, bundle)
}

// Import импортирует переносимый пакет в рабочее пространство.
func (s *UseCase) Import(ctx context.Context, planID, confirmation, ifMatch string) (*entities.SnapshotImportResult, error) {
	return s.coordinator.Import(ctx, planID, confirmation, ifMatch)
}
