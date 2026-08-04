package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type StyleRepository struct {
	store *EndgeRepository
}

func NewStyleRepository(store *EndgeRepository) ports.StyleRepository {
	return &StyleRepository{store: store}
}

func (r *StyleRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "styles", filter)
}

func (r *StyleRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "styles", identity, includeDeleted)
}

func (r *StyleRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *StyleRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.StyleRepository = (*StyleRepository)(nil)
