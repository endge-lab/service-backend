package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewI18nBundleRepository(store *EndgeRepository) ports.I18nBundleRepository {
	return newDocumentRepository(store, entities.CollectionI18nBundles, true)
}
