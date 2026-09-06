package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewDataViewRepository(store *EndgeRepository) ports.DataViewRepository {
	return newDocumentRepository(store, entities.CollectionDataViews, true)
}
