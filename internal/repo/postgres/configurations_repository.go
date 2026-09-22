package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewConfigurationRepository(store *EndgeRepository) ports.ConfigurationRepository {
	return newDocumentRepository(store, entities.CollectionConfigurations, false)
}
