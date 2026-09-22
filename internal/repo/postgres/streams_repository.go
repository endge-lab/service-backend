package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewStreamRepository(store *EndgeRepository) ports.StreamRepository {
	return newDocumentRepository(store, entities.CollectionStreams, true)
}
