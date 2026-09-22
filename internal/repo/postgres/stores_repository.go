package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewStoreRepository(store *EndgeRepository) ports.StoreRepository {
	return newDocumentRepository(store, entities.CollectionStores, true)
}
