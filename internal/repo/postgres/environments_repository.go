package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewEnvironmentRepository(store *EndgeRepository) ports.EnvironmentRepository {
	return newDocumentRepository(store, entities.CollectionEnvironments, true)
}
