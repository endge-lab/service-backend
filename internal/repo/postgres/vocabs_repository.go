package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type VocabRepository struct {
	store *EndgeRepository
}

func NewVocabRepository(store *EndgeRepository) ports.VocabRepository {
	return &VocabRepository{store: store}
}

func (r *VocabRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "vocabs", filter)
}

func (r *VocabRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "vocabs", identity, includeDeleted)
}

func (r *VocabRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *VocabRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.VocabRepository = (*VocabRepository)(nil)
