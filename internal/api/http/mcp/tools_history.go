package mcp

import (
	"context"

	"github.com/endge-lab/service-backend/internal/api/http/v1/commit"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type revisionInput struct {
	Collection    string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity      string `json:"identity" jsonschema:"Identity документа"`
	RevisionID    string `json:"revisionId" jsonschema:"id ревизии из endge_document_revisions или beforeRevisionId/afterRevisionId из endge_get_commit"`
	FacetIdentity string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Workspace     string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
}

type restoreRevisionInput struct {
	Collection    string `json:"collection" jsonschema:"Коллекция: документы (queries, components, data-views и др.), фасеты facets или документы фасета facet-documents"`
	Identity      string `json:"identity" jsonschema:"Identity документа"`
	RevisionID    string `json:"revisionId" jsonschema:"id ревизии, к которой откатить документ (endge_document_revisions)"`
	Revision      int    `json:"revision" jsonschema:"Текущая revision документа (поле revision из endge_get_document), как If-Match в REST"`
	FacetIdentity string `json:"facetIdentity,omitempty" jsonschema:"Только для collection=facet-documents: identity фасета (endge_list_documents с collection=facets)"`
	Workspace     string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
}

type listCommitsInput struct {
	Workspace string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Размер страницы, от новых коммитов к старым"`
	Offset    int    `json:"offset,omitempty" jsonschema:"Смещение"`
}

type commitIDInput struct {
	ID        string `json:"id" jsonschema:"id коммита из endge_list_commits или committedInCommitId ревизии"`
	Workspace string `json:"workspace,omitempty" jsonschema:"Identity workspace. С персональным токеном можно не передавать: используется workspace токена"`
}

// revisionView — ревизия для агента: без служебных полей (checksum, mutationBatchId), снимок по запросу.
func revisionView(revision entities.Revision, withSnapshot bool) map[string]any {
	item := map[string]any{
		"id": revision.ID, "collection": revision.DocumentType, "identity": revision.DocumentIdentity,
		"revisionNumber": revision.RevisionNumber, "operation": revision.Operation,
		"createdBy": revision.CreatedBy, "createdAt": revision.CreatedAt, "committedInCommitId": revision.CommittedInCommitID,
	}
	if revision.RestoredFromRevisionID != nil {
		item["restoredFromRevisionId"] = *revision.RestoredFromRevisionID
	}
	if revision.AccessTokenID != nil {
		item["accessTokenId"] = *revision.AccessTokenID
	}
	if revision.AccessTokenName != nil {
		item["accessTokenName"] = *revision.AccessTokenName
	}
	if withSnapshot {
		item["snapshot"] = revision.Snapshot
	}
	return item
}

func (t *tools) revisions(ctx context.Context, _ *mcp.CallToolRequest, in revisionsInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_document_revisions"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	options := ports.RevisionListOptions{IncludeSnapshot: in.WithSnapshot, Limit: in.Limit, Offset: in.Offset}
	if options.Limit == 0 {
		options.Limit = 20
	}
	page, err := store.revisions(ctx, facet, in.Identity, options)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, revision := range page.Items {
		items = append(items, revisionView(revision, in.WithSnapshot))
	}
	return jsonResult(map[string]any{"items": items, "total": page.Total, "limit": options.Limit, "offset": options.Offset})
}

func (t *tools) getRevision(ctx context.Context, _ *mcp.CallToolRequest, in revisionInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_get_revision"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	revision, err := store.revision(ctx, facet, in.Identity, in.RevisionID)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(revisionView(*revision, true))
}

// restoreRevision — откат к ревизии; это запись, поэтому инструмент есть только в наборе write,
// а роль, суженную токеном, и revision проверяет use case.
func (t *tools) restoreRevision(ctx context.Context, _ *mcp.CallToolRequest, in restoreRevisionInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_restore_revision"
	ctx, store, facet, err := t.documentTarget(ctx, in.Collection, in.FacetIdentity, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return t.writeSummary(tool, in.Collection, store, func() (*entities.Document, error) {
		return store.restoreRevision(ctx, facet, in.Identity, in.RevisionID, in.Revision)
	})
}

func (t *tools) listCommits(ctx context.Context, _ *mcp.CallToolRequest, in listCommitsInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_list_commits"
	ctx, err := t.workspaceContext(ctx, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	values, err := t.useCases.Commits.List(ctx)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	page := pageOf(values, limit, in.Offset)
	items := make([]commit.Response, 0, len(page))
	for _, value := range page {
		value.Changes = nil
		response, mapErr := commit.NewResponse(value)
		if mapErr != nil {
			return errorResult(t.logger, tool, mapErr)
		}
		items = append(items, response)
	}
	return jsonResult(map[string]any{"items": items, "total": len(values), "limit": limit, "offset": in.Offset})
}

func (t *tools) getCommit(ctx context.Context, _ *mcp.CallToolRequest, in commitIDInput) (*mcp.CallToolResult, any, error) {
	const tool = "endge_get_commit"
	ctx, err := t.workspaceContext(ctx, in.Workspace)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	value, err := t.useCases.Commits.Get(ctx, in.ID)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	response, err := commit.NewResponse(*value)
	if err != nil {
		return errorResult(t.logger, tool, err)
	}
	return jsonResult(response)
}
