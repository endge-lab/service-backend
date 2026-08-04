package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type I18nBundleRepository struct {
	store *EndgeRepository
}

func NewI18nBundleRepository(store *EndgeRepository) ports.I18nBundleRepository {
	return &I18nBundleRepository{store: store}
}

func (r *I18nBundleRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "i18n-bundles", filter)
}

func (r *I18nBundleRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "i18n-bundles", identity, includeDeleted)
}

func (r *I18nBundleRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *I18nBundleRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.I18nBundleRepository = (*I18nBundleRepository)(nil)
