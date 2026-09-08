package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewTypeRepository(store *EndgeRepository) ports.TypeRepository {
	return newDocumentRepository(store, entities.CollectionTypes, true)
}
