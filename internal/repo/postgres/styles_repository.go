package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewStyleRepository(store *EndgeRepository) ports.StyleRepository {
	return newDocumentRepository(store, entities.CollectionStyles, true)
}
