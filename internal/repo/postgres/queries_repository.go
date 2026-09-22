package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewQueryRepository(store *EndgeRepository) ports.QueryRepository {
	return newDocumentRepository(store, entities.CollectionQueries, true)
}
