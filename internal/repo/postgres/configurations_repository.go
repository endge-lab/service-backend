package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type ConfigurationRepository struct{ store *EndgeRepository }

func NewConfigurationRepository(store *EndgeRepository) ports.ConfigurationRepository {
	return &ConfigurationRepository{store: store}
}

func (r *ConfigurationRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "configurations", filter)
}

func (r *ConfigurationRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "configurations", identity, includeDeleted)
}

func (r *ConfigurationRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, nil)
}

func (r *ConfigurationRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, nil)
}

var _ ports.ConfigurationRepository = (*ConfigurationRepository)(nil)
