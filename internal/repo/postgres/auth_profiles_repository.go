package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewAuthProfileRepository(store *EndgeRepository) ports.AuthProfileRepository {
	return newDocumentRepository(store, entities.CollectionAuthProfiles, true)
}
