package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type ProjectRepository struct {
	store *EndgeRepository
}

func NewProjectRepository(store *EndgeRepository) ports.ProjectRepository {
	return &ProjectRepository{store: store}
}

func (r *ProjectRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "projects", filter)
}

func (r *ProjectRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "projects", identity, includeDeleted)
}

func (r *ProjectRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *ProjectRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.ProjectRepository = (*ProjectRepository)(nil)
