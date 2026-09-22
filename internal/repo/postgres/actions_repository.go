package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewActionRepository(store *EndgeRepository) ports.ActionRepository {
	return newDocumentRepository(store, entities.CollectionActions, true)
}
