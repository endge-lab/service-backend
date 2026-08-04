package commits

import (
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
)

// UseCase координирует сценарии работы с коммитами рабочего пространства.
type UseCase struct {
	repository  ports.CommitRepository
	revisions   ports.RevisionRepository
	tx          ports.TxManager
	coordinator *workspace_state.Coordinator
}

// NewUseCase создаёт use case для работы с коммитами рабочего пространства.
func NewUseCase(repository ports.CommitRepository, revisions ports.RevisionRepository, tx ports.TxManager, coordinator *workspace_state.Coordinator) *UseCase {
	return &UseCase{repository: repository, revisions: revisions, tx: tx, coordinator: coordinator}
}
