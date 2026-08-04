package workspaces

import (
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// UseCase координирует сценарии работы с рабочими пространствами.
type UseCase struct {
	workspaces ports.WorkspaceRepository
	documents  ports.DocumentRepository
	commits    ports.CommitRepository
	tx         ports.TxManager
	history    *history.Recorder
}

// NewUseCase создаёт use case для работы с рабочими пространствами.
func NewUseCase(workspaces ports.WorkspaceRepository, documents ports.DocumentRepository, commits ports.CommitRepository, tx ports.TxManager, recorder *history.Recorder) *UseCase {
	return &UseCase{workspaces: workspaces, documents: documents, commits: commits, tx: tx, history: recorder}
}
