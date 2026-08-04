// Package backups владеет ручными и автоматическими snapshot backups рабочего пространства.
package backups

import "github.com/endge-lab/service-backend/internal/usecase/ports"

// UseCase координирует создание и чтение backups.
type UseCase struct {
	backups  ports.SnapshotRepository
	portable ports.PortableRepository
	tx       ports.TxManager
}

// NewUseCase создаёт use case backups.
func NewUseCase(backups ports.SnapshotRepository, portable ports.PortableRepository, tx ports.TxManager) *UseCase {
	return &UseCase{backups: backups, portable: portable, tx: tx}
}
