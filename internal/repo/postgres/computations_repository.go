package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewComputationRepository(store *EndgeRepository) ports.ComputationRepository {
	return newDocumentRepository(store, entities.CollectionComputations, true)
}
