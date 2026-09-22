package document_move

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/documents"
)

// UseCase задаёт application-контракт массового перемещения документов.
type UseCase interface {
	MoveDocuments(context.Context, documents.MoveDocumentsInput) (documents.MoveDocumentsResult, error)
}

// BindUseCase предоставляет document lifecycle как HTTP-порт массового перемещения.
func BindUseCase(lifecycle *documents.Lifecycle) UseCase { return lifecycle }
