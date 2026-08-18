package portable

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// Status returns the current committed domain identity and dirty state.
func (s *UseCase) Status(ctx context.Context) (*entities.DomainStatus, error) {
	return s.coordinator.Status(ctx)
}

// Export экспортирует состояние рабочего пространства в переносимый пакет.
func (s *UseCase) Export(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.Export(ctx)
}

// Live возвращает полный рабочий snapshot с локальными state-полями для Configurator.
func (s *UseCase) Live(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.ExportLive(ctx)
}
