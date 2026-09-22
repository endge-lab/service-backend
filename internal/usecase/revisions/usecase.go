package revisions

import (
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
)

// UseCase координирует сценарии работы с ревизиями документов.
type UseCase struct {
	repository  ports.RevisionRepository
	coordinator *workspace_state.Coordinator
}

// NewUseCase создаёт use case для работы с ревизиями документов.
func NewUseCase(repository ports.RevisionRepository, coordinator *workspace_state.Coordinator) *UseCase {
	return &UseCase{repository: repository, coordinator: coordinator}
}
