package portable

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
)

// Status returns the current committed domain identity and dirty state.
func (s *UseCase) Status(ctx context.Context) (*entities.DomainStatus, error) {
	return s.coordinator.Status(ctx)
}

// Export экспортирует состояние рабочего пространства в переносимый пакет.
func (s *UseCase) Export(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.Export(ctx)
}

// ExportWithOptions includes explicitly selected operational data and optionally encrypts the whole artifact.
func (s *UseCase) ExportWithOptions(ctx context.Context, options workspace_state.ExportOptions, password string) (json.RawMessage, error) {
	raw, err := s.coordinator.ExportWithOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	return encryptArtifact(raw, password)
}

// Live возвращает полный рабочий snapshot с локальными state-полями для Configurator.
func (s *UseCase) Live(ctx context.Context) (json.RawMessage, error) {
	return s.coordinator.ExportLive(ctx)
}
