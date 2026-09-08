package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewConverterRepository(store *EndgeRepository) ports.ConverterRepository {
	return newDocumentRepository(store, entities.CollectionConverters, true)
}
