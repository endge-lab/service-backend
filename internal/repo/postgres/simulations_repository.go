package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewSimulationRepository(store *EndgeRepository) ports.SimulationRepository {
	return newDocumentRepository(store, entities.CollectionSimulations, true)
}
