package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewComponentRepository(store *EndgeRepository) ports.ComponentRepository {
	return newDocumentRepository(store, entities.CollectionComponents, true)
}
