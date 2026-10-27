package postgres

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

func (r *EndgeRepository) NextWorkspaceSequence(ctx context.Context, workspaceID string) (int64, error) {
	var value int64
	err := r.executor(ctx).QueryRow(ctx, `UPDATE workspaces SET head_sequence=head_sequence+1,updated_at=NOW() WHERE id=$1 RETURNING head_sequence`, workspaceID).Scan(&value)
	return value, err
}

// CreateMutationBatch помечает batch токеном запроса, если изменение пришло от агента.
// Отметка берётся из context, поэтому её получают все пути записи без изменения сигнатур use case.
func (r *EndgeRepository) CreateMutationBatch(ctx context.Context, workspaceID *string, kind, actor string) (string, error) {
	id := uuid.NewString()
	var accessTokenID *string
	if principal, ok := entities.AccessTokenFromContext(ctx); ok {
		accessTokenID = &principal.TokenID
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO mutation_batches(id,workspace_id,kind,actor_user_id,access_token_id) VALUES($1,$2,$3,$4,$5)`, id, workspaceID, kind, actor, accessTokenID)
	return id, err
}
func (r *EndgeRepository) LatestRevision(ctx context.Context, workspaceID *string, kind, documentID string) (*entities.Revision, error) {
	row := r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.workspace_id IS NOT DISTINCT FROM $1 AND r.document_type=$2 AND r.document_id=$3 ORDER BY r.revision_number DESC LIMIT 1`, workspaceID, kind, documentID)
	return scanRevision(row)
}
func revisionSelect() string { return revisionSelectWith(true) }

// revisionSelectWith позволяет не читать снимки: в истории без snapshot они заменяются null.
func revisionSelectWith(includeSnapshot bool) string {
	snapshot := "r.snapshot"
	if !includeSnapshot {
		snapshot = "'null'::jsonb"
	}
	return `SELECT r.id::text,COALESCE(r.workspace_id::text,''),r.document_type,r.document_id::text,r.document_identity,r.revision_number,r.workspace_sequence,r.operation,r.parent_revision_id::text,r.restored_from_revision_id::text,r.committed_in_commit_id::text,r.mutation_batch_id::text,r.snapshot_version,` + snapshot + `,r.checksum,` + actorScan("u") + `,
		(SELECT mb.access_token_id::text FROM mutation_batches mb WHERE mb.id=r.mutation_batch_id),
		(SELECT t.name FROM mutation_batches mb JOIN access_tokens t ON t.id=mb.access_token_id WHERE mb.id=r.mutation_batch_id),
		COALESCE((SELECT jsonb_agg(` + actorScan("contributor") + ` ORDER BY contributor.username,contributor.id)
			FROM service_users contributor WHERE contributor.id=ANY(r.contributor_user_ids)),'[]'::jsonb),r.created_at
		FROM document_revisions r JOIN service_users u ON u.id=r.created_by`
}
func scanRevision(row scanner) (*entities.Revision, error) {
	v := &entities.Revision{}
	var workspace string
	var created, contributors []byte
	if err := row.Scan(&v.ID, &workspace, &v.DocumentType, &v.DocumentID, &v.DocumentIdentity, &v.RevisionNumber, &v.WorkspaceSequence, &v.Operation, &v.ParentRevisionID, &v.RestoredFromRevisionID, &v.CommittedInCommitID, &v.MutationBatchID, &v.SnapshotVersion, &v.Snapshot, &v.Checksum, &created, &v.AccessTokenID, &v.AccessTokenName, &contributors, &v.CreatedAt); err != nil {
		return nil, repositoryError(err)
	}
	v.WorkspaceID = workspace
	_ = json.Unmarshal(created, &v.CreatedBy)
	_ = json.Unmarshal(contributors, &v.Contributors)
	for _, contributor := range v.Contributors {
		v.ContributorUserIDs = append(v.ContributorUserIDs, contributor.ID)
	}
	return v, nil
}
func (r *EndgeRepository) InsertRevision(ctx context.Context, v entities.Revision) (*entities.Revision, error) {
	var workspace any
	if v.WorkspaceID != "" {
		workspace = v.WorkspaceID
	}
	contributorUserIDs := v.ContributorUserIDs
	if contributorUserIDs == nil {
		contributorUserIDs = []string{}
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO document_revisions(id,workspace_id,document_type,document_id,document_identity,revision_number,workspace_sequence,operation,parent_revision_id,restored_from_revision_id,mutation_batch_id,snapshot_version,snapshot,checksum,created_by,contributor_user_ids) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,ARRAY(SELECT value::uuid FROM unnest($16::text[]) value))`, v.ID, workspace, v.DocumentType, v.DocumentID, v.DocumentIdentity, v.RevisionNumber, v.WorkspaceSequence, v.Operation, v.ParentRevisionID, v.RestoredFromRevisionID, v.MutationBatchID, v.SnapshotVersion, v.Snapshot, v.Checksum, v.CreatedBy.ID, contributorUserIDs)
	if err != nil {
		return nil, err
	}
	return scanRevision(r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.id=$1`, v.ID))
}

// ListRevisions возвращает страницу истории документа от новых ревизий к старым и общее число ревизий.
func (r *EndgeRepository) ListRevisions(ctx context.Context, workspaceID, kind, identity string, options ports.RevisionListOptions) (ports.RevisionPage, error) {
	page := ports.RevisionPage{Items: []entities.Revision{}}
	document, err := r.GetDocument(ctx, workspaceID, kind, identity, true)
	if err != nil {
		return page, err
	}
	if err = r.executor(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM document_revisions WHERE workspace_id=$1 AND document_type=$2 AND document_id=$3`, workspaceID, kind, document.ID).Scan(&page.Total); err != nil {
		return page, err
	}
	// LIMIT NULL в PostgreSQL означает «без ограничения».
	var limit any
	if options.Limit > 0 {
		limit = options.Limit
	}
	rows, err := r.executor(ctx).Query(ctx, revisionSelectWith(options.IncludeSnapshot)+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 ORDER BY r.revision_number DESC LIMIT $4 OFFSET $5`, workspaceID, kind, document.ID, limit, options.Offset)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		v, scanErr := scanRevision(rows)
		if scanErr != nil {
			return page, scanErr
		}
		page.Items = append(page.Items, *v)
	}
	return page, rows.Err()
}
func (r *EndgeRepository) GetRevision(ctx context.Context, workspaceID, kind, identity, id string) (*entities.Revision, error) {
	document, err := r.GetDocument(ctx, workspaceID, kind, identity, true)
	if err != nil {
		return nil, err
	}
	return scanRevision(r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 AND r.id=$4`, workspaceID, kind, document.ID, id))
}

func (r *EndgeRepository) listRevisionsByDocumentID(ctx context.Context, workspaceID, kind, documentID string) ([]entities.Revision, error) {
	rows, err := r.executor(ctx).Query(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 ORDER BY r.revision_number DESC`, workspaceID, kind, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Revision{}
	for rows.Next() {
		value, scanErr := scanRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) getRevisionByDocumentID(ctx context.Context, workspaceID, kind, documentID, revisionID string) (*entities.Revision, error) {
	return scanRevision(r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 AND r.id=$4`, workspaceID, kind, documentID, revisionID))
}

func (r *EndgeRepository) ListFacetRevisions(ctx context.Context, workspaceID, identity string) ([]entities.Revision, error) {
	document, err := r.GetFacet(ctx, workspaceID, identity, true)
	if err != nil {
		return nil, err
	}
	return r.listRevisionsByDocumentID(ctx, workspaceID, entities.CollectionFacets, document.ID)
}

func (r *EndgeRepository) GetFacetRevision(ctx context.Context, workspaceID, identity, revisionID string) (*entities.Revision, error) {
	document, err := r.GetFacet(ctx, workspaceID, identity, true)
	if err != nil {
		return nil, err
	}
	return r.getRevisionByDocumentID(ctx, workspaceID, entities.CollectionFacets, document.ID, revisionID)
}

func (r *EndgeRepository) ListFacetDocumentRevisions(ctx context.Context, workspaceID, facetIdentity, identity string) ([]entities.Revision, error) {
	document, err := r.GetFacetDocument(ctx, workspaceID, facetIdentity, identity, true)
	if err != nil {
		return nil, err
	}
	return r.listRevisionsByDocumentID(ctx, workspaceID, entities.CollectionFacetDocuments, document.ID)
}

func (r *EndgeRepository) GetFacetDocumentRevision(ctx context.Context, workspaceID, facetIdentity, identity, revisionID string) (*entities.Revision, error) {
	document, err := r.GetFacetDocument(ctx, workspaceID, facetIdentity, identity, true)
	if err != nil {
		return nil, err
	}
	return r.getRevisionByDocumentID(ctx, workspaceID, entities.CollectionFacetDocuments, document.ID, revisionID)
}
