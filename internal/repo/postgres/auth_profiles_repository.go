package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type AuthProfileRepository struct {
	store *EndgeRepository
}

func NewAuthProfileRepository(store *EndgeRepository) ports.AuthProfileRepository {
	return &AuthProfileRepository{store: store}
}

func (r *AuthProfileRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, "auth-profiles", filter)
}

func (r *AuthProfileRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, "auth-profiles", identity, includeDeleted)
}

func (r *AuthProfileRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, folderID)
}

func (r *AuthProfileRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, folderID)
}

var _ ports.AuthProfileRepository = (*AuthProfileRepository)(nil)
