package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type EnvironmentRepository struct {
	store *EndgeRepository
}

func NewEnvironmentRepository(store *EndgeRepository) ports.EnvironmentRepository {
	return &EnvironmentRepository{store: store}
}

func (r *EnvironmentRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "environments", filter)
}

func (r *EnvironmentRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "environments", identity, includeDeleted)
}

func (r *EnvironmentRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *EnvironmentRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.EnvironmentRepository = (*EnvironmentRepository)(nil)
