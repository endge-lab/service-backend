package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewFolderRepository(store *EndgeRepository) ports.FolderRepository {
	return newDocumentRepository(store, entities.CollectionFolders, true)
}
