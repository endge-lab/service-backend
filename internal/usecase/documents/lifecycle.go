package documents

import (
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// Definition закрепляет коллекцию за конкретным ресурсным use case.
// Благодаря этому HTTP-транспорт не может выбирать произвольную таблицу репозитория.
type Definition struct {
	Collection string
}

// Lifecycle координирует единый жизненный цикл документов, ревизий и связей.
type Lifecycle struct {
	documents ports.DocumentRepository
	tx        ports.TxManager
	history   *history.Recorder
}

// NewLifecycle создаёт сервис жизненного цикла документов.
func NewLifecycle(documents ports.DocumentRepository, tx ports.TxManager, recorder *history.Recorder) *Lifecycle {
	return &Lifecycle{documents: documents, tx: tx, history: recorder}
}
