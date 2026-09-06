package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewMockRepository(store *EndgeRepository) ports.MockRepository {
	return newDocumentRepository(store, entities.CollectionMocks, true)
}
