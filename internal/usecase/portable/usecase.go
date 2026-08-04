package portable

import "github.com/endge-lab/service-backend/internal/usecase/workspace_state"

// UseCase координирует сценарии работы с переносимыми пакетами.
type UseCase struct{ coordinator *workspace_state.Coordinator }

// NewUseCase создаёт use case для работы с переносимыми пакетами.
func NewUseCase(coordinator *workspace_state.Coordinator) *UseCase {
	return &UseCase{coordinator: coordinator}
}
