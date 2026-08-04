package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type NavigationRepository struct {
	store *EndgeRepository
}

func NewNavigationRepository(store *EndgeRepository) ports.NavigationRepository {
	return &NavigationRepository{store: store}
}

func (r *NavigationRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "navigations", filter)
}

func (r *NavigationRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "navigations", identity, includeDeleted)
}

func (r *NavigationRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *NavigationRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.NavigationRepository = (*NavigationRepository)(nil)
