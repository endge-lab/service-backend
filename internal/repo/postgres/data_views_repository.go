package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type DataViewRepository struct {
	store *EndgeRepository
}

func NewDataViewRepository(store *EndgeRepository) ports.DataViewRepository {
	return &DataViewRepository{store: store}
}

func (r *DataViewRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "data-views", filter)
}

func (r *DataViewRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "data-views", identity, includeDeleted)
}

func (r *DataViewRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *DataViewRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.DataViewRepository = (*DataViewRepository)(nil)
