package postgres

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func NewProjectRepository(store *EndgeRepository) ports.ProjectRepository {
	return newDocumentRepository(store, entities.CollectionProjects, true)
}
