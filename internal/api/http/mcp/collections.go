package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/api/http/v1/facet"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// toolCollections — коллекции инструментов документов: документное API и фасеты.
// Фасеты (facets) и документы фасетов (facet-documents) живут вне документного API,
// но агент работает с ними теми же инструментами; facet-documents требуют facetIdentity.
var toolCollections = append(append([]string(nil), entities.DocumentCollections...), entities.FacetCollections...)

// collectionStore — операции одной коллекции для инструментов MCP. Реализации вызывают
// те же use case и DTO, что REST-ресурс коллекции; facet — identity фасета для facet-documents.
type collectionStore interface {
	list(ctx context.Context, facet string, filter ports.DocumentFilter) ([]entities.Document, error)
	get(ctx context.Context, facet, identity string, includeDeleted bool) (*entities.Document, error)
	create(ctx context.Context, facet string, payload []byte) (*entities.Document, error)
	patch(ctx context.Context, facet, identity string, payload []byte, expected int) (*entities.Document, error)
	remove(ctx context.Context, facet, identity string, expected int) (*entities.Document, error)
	restore(ctx context.Context, facet, identity string, expected int) (*entities.Document, error)
	revisions(ctx context.Context, facet, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error)
	revision(ctx context.Context, facet, identity, revisionID string) (*entities.Revision, error)
	restoreRevision(ctx context.Context, facet, identity, revisionID string, expected int) (*entities.Document, error)
	view(entities.Document) (map[string]any, error)
}

// store выбирает реализацию коллекции. Неизвестная коллекция — та же ошибка, что в REST;
// facet-documents без facetIdentity — facet_identity_required.
func (t *tools) store(collection, facetIdentity string) (collectionStore, string, error) {
	facetIdentity = strings.TrimSpace(facetIdentity)
	switch collection {
	case entities.CollectionFacets:
		return facetStore{useCase: t.useCases.Facets, contracts: t.contracts.Facets}, "", nil
	case entities.CollectionFacetDocuments:
		if facetIdentity == "" {
			return nil, "", domainerrors.InvalidInput("facet_identity_required", "facetIdentity is required for facet-documents")
		}
		return facetDocumentStore{useCase: t.useCases.Facets, contracts: t.contracts.Facets}, facetIdentity, nil
	}
	useCase, ok := t.useCases.Documents[collection]
	if !ok {
		return nil, "", domainerrors.WithDetails(domainerrors.InvalidInput("collection_unsupported", "Collection is not supported"), map[string]any{"collection": collection})
	}
	return documentStore{collection: collection, useCase: useCase, contract: t.documentContract(collection), revisionUseCase: t.useCases.Revisions}, "", nil
}

// documentStore — коллекция документного API (/api/v1/<коллекция>) и история
// /api/v1/domain/documents/{type}/{identity}/revisions.
type documentStore struct {
	collection      string
	useCase         DocumentUseCase
	contract        shared.DocumentContract
	revisionUseCase RevisionUseCase
}

func (s documentStore) list(ctx context.Context, _ string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.useCase.List(ctx, filter)
}

func (s documentStore) get(ctx context.Context, _, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.useCase.Get(ctx, identity, includeDeleted)
}

func (s documentStore) create(ctx context.Context, _ string, payload []byte) (*entities.Document, error) {
	if err := s.contract.Create.Validate(payload); err != nil {
		return nil, err
	}
	input, err := documents.NewCreateInputJSON(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.Create(ctx, input)
}

func (s documentStore) patch(ctx context.Context, _, identity string, payload []byte, expected int) (*entities.Document, error) {
	if err := s.contract.Patch.Validate(payload); err != nil {
		return nil, err
	}
	input, err := documents.NewPatchInputJSON(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.Patch(ctx, identity, input, expected)
}

func (s documentStore) remove(ctx context.Context, _, identity string, expected int) (*entities.Document, error) {
	return s.useCase.Delete(ctx, identity, expected)
}

func (s documentStore) restore(ctx context.Context, _, identity string, expected int) (*entities.Document, error) {
	return s.useCase.Restore(ctx, identity, expected)
}

func (s documentStore) revisions(ctx context.Context, _, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	return s.revisionUseCase.List(ctx, s.collection, identity, options)
}

func (s documentStore) revision(ctx context.Context, _, identity, revisionID string) (*entities.Revision, error) {
	return s.revisionUseCase.Get(ctx, s.collection, identity, revisionID)
}

func (s documentStore) restoreRevision(ctx context.Context, _, identity, revisionID string, expected int) (*entities.Document, error) {
	return s.revisionUseCase.Restore(ctx, s.collection, identity, revisionID, expected)
}

func (documentStore) view(document entities.Document) (map[string]any, error) {
	return documentView(document)
}

// facetStore — фасеты /api/v1/facets.
type facetStore struct {
	useCase   FacetUseCase
	contracts FacetContracts
}

func (s facetStore) list(ctx context.Context, _ string, filter ports.DocumentFilter) ([]entities.Document, error) {
	values, err := s.useCase.List(ctx, filter.IncludeDeleted)
	return pageOf(values, filter.Limit, filter.Offset), err
}

func (s facetStore) get(ctx context.Context, _, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.useCase.Get(ctx, identity, includeDeleted)
}

func (s facetStore) create(ctx context.Context, _ string, payload []byte) (*entities.Document, error) {
	input, err := s.contracts.Create(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.Create(ctx, input)
}

func (s facetStore) patch(ctx context.Context, _, identity string, payload []byte, expected int) (*entities.Document, error) {
	input, err := s.contracts.Patch(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.Patch(ctx, identity, input, expected)
}

func (s facetStore) remove(ctx context.Context, _, identity string, expected int) (*entities.Document, error) {
	return s.useCase.Delete(ctx, identity, expected)
}

func (s facetStore) restore(ctx context.Context, _, identity string, expected int) (*entities.Document, error) {
	return s.useCase.Restore(ctx, identity, expected)
}

func (s facetStore) revisions(ctx context.Context, _, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	values, err := s.useCase.ListRevisions(ctx, identity)
	return revisionPage(values, options), err
}

func (s facetStore) revision(ctx context.Context, _, identity, revisionID string) (*entities.Revision, error) {
	return s.useCase.GetRevision(ctx, identity, revisionID)
}

func (s facetStore) restoreRevision(ctx context.Context, _, identity, revisionID string, expected int) (*entities.Document, error) {
	return s.useCase.RestoreRevision(ctx, identity, revisionID, expected)
}

func (facetStore) view(document entities.Document) (map[string]any, error) {
	response, err := facet.NewFacetResponse(document)
	if err != nil {
		return nil, err
	}
	return objectOf(response)
}

// facetDocumentStore — документы фасета /api/v1/facets/{facetIdentity}/documents.
type facetDocumentStore struct {
	useCase   FacetUseCase
	contracts FacetContracts
}

func (s facetDocumentStore) list(ctx context.Context, facetIdentity string, filter ports.DocumentFilter) ([]entities.Document, error) {
	return s.useCase.ListDocuments(ctx, facetIdentity, filter)
}

func (s facetDocumentStore) get(ctx context.Context, facetIdentity, identity string, includeDeleted bool) (*entities.Document, error) {
	return s.useCase.GetDocument(ctx, facetIdentity, identity, includeDeleted)
}

func (s facetDocumentStore) create(ctx context.Context, facetIdentity string, payload []byte) (*entities.Document, error) {
	input, err := s.contracts.DocumentCreate(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.CreateDocument(ctx, facetIdentity, input)
}

func (s facetDocumentStore) patch(ctx context.Context, facetIdentity, identity string, payload []byte, expected int) (*entities.Document, error) {
	input, err := s.contracts.DocumentPatch(payload)
	if err != nil {
		return nil, err
	}
	return s.useCase.PatchDocument(ctx, facetIdentity, identity, input, expected)
}

func (s facetDocumentStore) remove(ctx context.Context, facetIdentity, identity string, expected int) (*entities.Document, error) {
	return s.useCase.DeleteDocument(ctx, facetIdentity, identity, expected)
}

func (s facetDocumentStore) restore(ctx context.Context, facetIdentity, identity string, expected int) (*entities.Document, error) {
	return s.useCase.RestoreDocument(ctx, facetIdentity, identity, expected)
}

func (s facetDocumentStore) revisions(ctx context.Context, facetIdentity, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	values, err := s.useCase.ListDocumentRevisions(ctx, facetIdentity, identity)
	return revisionPage(values, options), err
}

func (s facetDocumentStore) revision(ctx context.Context, facetIdentity, identity, revisionID string) (*entities.Revision, error) {
	return s.useCase.GetDocumentRevision(ctx, facetIdentity, identity, revisionID)
}

func (s facetDocumentStore) restoreRevision(ctx context.Context, facetIdentity, identity, revisionID string, expected int) (*entities.Document, error) {
	return s.useCase.RestoreDocumentRevision(ctx, facetIdentity, identity, revisionID, expected)
}

func (facetDocumentStore) view(document entities.Document) (map[string]any, error) {
	response, err := facet.NewFacetDocumentResponse(document)
	if err != nil {
		return nil, err
	}
	return objectOf(response)
}

// pageOf применяет limit/offset к списку, который REST отдаёт целиком (фасеты, их история).
func pageOf[T any](values []T, limit, offset int) []T {
	offset = max(offset, 0)
	if offset >= len(values) {
		return []T{}
	}
	values = values[offset:]
	if limit > 0 && limit < len(values) {
		values = values[:limit]
	}
	return values
}

// revisionPage — страница истории фасета: total — вся история, снимки по запросу.
func revisionPage(values []entities.Revision, options ports.RevisionListOptions) ports.RevisionPage {
	return ports.RevisionPage{Items: pageOf(values, options.Limit, options.Offset), Total: len(values)}
}

// objectOf переводит REST-ответ в JSON-объект без typed nil.
func objectOf(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	return result, json.Unmarshal(raw, &result)
}
