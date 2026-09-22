package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewCompositionRepository(store *EndgeRepository) ports.CompositionRepository {
	return newDocumentRepository(store, entities.CollectionCompositions, true)
}
