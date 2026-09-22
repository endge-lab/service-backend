package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewVocabRepository(store *EndgeRepository) ports.VocabRepository {
	return newDocumentRepository(store, entities.CollectionVocabs, true)
}
