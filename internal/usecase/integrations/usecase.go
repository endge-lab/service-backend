package integrations

import (
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// UseCase координирует сценарии работы с интеграциями.
type UseCase struct {
	repository ports.IntegrationRepository
	tx         ports.TxManager
	history    *history.Recorder
}

// NewUseCase создаёт use case для работы с интеграциями.
func NewUseCase(repository ports.IntegrationRepository, tx ports.TxManager, recorder *history.Recorder) *UseCase {
	return &UseCase{repository: repository, tx: tx, history: recorder}
}
