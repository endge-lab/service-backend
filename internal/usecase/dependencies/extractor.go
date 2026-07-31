package dependencies

import "github.com/endge-lab/service-backend/internal/domain/entities"

// DependencyExtractionResult содержит результат typed разбора одного
// canonical document. Unverified result разрешает сохранить work-in-progress,
// но явно запрещает считать его dependencies полностью проверенными.
type DependencyExtractionResult struct {
	References        []entities.DomainDependencyReference
	VerificationState entities.DomainDependencyVerificationState
	VerificationError *string
}

// DependencyExtractor разбирает document конкретной entity-модели без generic
// reflection. Тип T связывает extractor с его canonical document на этапе
// компиляции.
type DependencyExtractor[T any] interface {
	Extract(document T) (DependencyExtractionResult, error)
}
