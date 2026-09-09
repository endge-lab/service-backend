package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// documentRepository adapts EndgeRepository operations to one document collection.
type documentRepository struct {
	store          *EndgeRepository
	collection     string
	supportsFolder bool
}

func newDocumentRepository(store *EndgeRepository, collection string, supportsFolder bool) *documentRepository {
	return &documentRepository{
		store:          store,
		collection:     collection,
		supportsFolder: supportsFolder,
	}
}

func (r *documentRepository) List(ctx context.Context, workspaceID string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return r.store.ListDocuments(ctx, workspaceID, r.collection, filter)
}

func (r *documentRepository) Get(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	return r.store.GetDocument(ctx, workspaceID, r.collection, identity, includeDeleted)
}

func (r *documentRepository) Insert(ctx context.Context, value entities.Document, folderID *string) (*entities.Document, error) {
	return r.store.InsertDocument(ctx, value, r.documentFolderID(folderID))
}

func (r *documentRepository) Update(ctx context.Context, value entities.Document, expectedRevision int, folderID *string) (*entities.Document, error) {
	return r.store.UpdateDocument(ctx, value, expectedRevision, r.documentFolderID(folderID))
}

func (r *documentRepository) documentFolderID(folderID *string) *string {
	if !r.supportsFolder {
		return nil
	}

	return folderID
}

var (
	_ ports.ActionRepository        = (*documentRepository)(nil)
	_ ports.AuthProfileRepository   = (*documentRepository)(nil)
	_ ports.ComponentRepository     = (*documentRepository)(nil)
	_ ports.CompositionRepository   = (*documentRepository)(nil)
	_ ports.ComputationRepository   = (*documentRepository)(nil)
	_ ports.ConfigurationRepository = (*documentRepository)(nil)
	_ ports.ConverterRepository     = (*documentRepository)(nil)
	_ ports.DataViewRepository      = (*documentRepository)(nil)
	_ ports.EnvironmentRepository   = (*documentRepository)(nil)
	_ ports.FilterRepository        = (*documentRepository)(nil)
	_ ports.FolderRepository        = (*documentRepository)(nil)
	_ ports.I18nBundleRepository    = (*documentRepository)(nil)
	_ ports.MockRepository          = (*documentRepository)(nil)
	_ ports.NavigationRepository    = (*documentRepository)(nil)
	_ ports.ProjectRepository       = (*documentRepository)(nil)
	_ ports.QueryRepository         = (*documentRepository)(nil)
	_ ports.StoreRepository         = (*documentRepository)(nil)
	_ ports.StreamRepository        = (*documentRepository)(nil)
	_ ports.SimulationRepository    = (*documentRepository)(nil)
	_ ports.StyleRepository         = (*documentRepository)(nil)
	_ ports.TenantRepository        = (*documentRepository)(nil)
	_ ports.TypeRepository          = (*documentRepository)(nil)
	_ ports.UpdateRepository        = (*documentRepository)(nil)
	_ ports.VocabRepository         = (*documentRepository)(nil)
)
