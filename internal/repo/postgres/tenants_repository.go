package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewTenantRepository(store *EndgeRepository) ports.TenantRepository {
	return newDocumentRepository(store, entities.CollectionTenants, true)
}
