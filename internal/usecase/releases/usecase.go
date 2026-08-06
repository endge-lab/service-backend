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
	artifacts   ports.ReleaseArtifactReader
}

// NewUseCase создаёт use case для работы с релизами рабочего пространства.
func NewUseCase(releases ports.ReleaseRepository, commits ports.CommitRepository, portable ports.PortableRepository, coordinator *workspace_state.Coordinator, artifacts ports.ReleaseArtifactReader) *UseCase {
	return &UseCase{releases: releases, commits: commits, portable: portable, coordinator: coordinator, artifacts: artifacts}
}
