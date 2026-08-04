package portable

import (
	"context"
	"encoding/json"
)

// Export экспортирует состояние рабочего пространства в переносимый пакет.
func (s *UseCase) Export(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.Export(ctx)
}

// Live возвращает полный рабочий snapshot с локальными state-полями для Configurator.
func (s *UseCase) Live(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.ExportLive(ctx)
}
