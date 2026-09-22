package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewFilterRepository(store *EndgeRepository) ports.FilterRepository {
	return newDocumentRepository(store, entities.CollectionFilters, true)
}
