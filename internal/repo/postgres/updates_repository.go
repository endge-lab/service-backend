package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewUpdateRepository(store *EndgeRepository) ports.UpdateRepository {
	return newDocumentRepository(store, entities.CollectionUpdates, true)
}
