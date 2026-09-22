package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewNavigationRepository(store *EndgeRepository) ports.NavigationRepository {
	return newDocumentRepository(store, entities.CollectionNavigations, true)
}
