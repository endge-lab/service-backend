package releases

import (
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
)

// UseCase координирует сценарии работы с релизами рабочего пространства.
type UseCase struct {
	releases    ports.ReleaseRepository
	commits     ports.CommitRepository
	portable    ports.PortableRepository
	coordinator *workspace_state.Coordinator
}

// NewUseCase создаёт use case для работы с релизами рабочего пространства.
func NewUseCase(releases ports.ReleaseRepository, commits ports.CommitRepository, portable ports.PortableRepository, coordinator *workspace_state.Coordinator) *UseCase {
	return &UseCase{releases: releases, commits: commits, portable: portable, coordinator: coordinator}
}
