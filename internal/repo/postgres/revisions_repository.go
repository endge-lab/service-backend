package postgres

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
)

func (r *EndgeRepository) NextWorkspaceSequence(ctx context.Context, workspaceID string) (int64, error) {
	var value int64
	err := r.executor(ctx).QueryRow(ctx, `UPDATE workspaces SET head_sequence=head_sequence+1,updated_at=NOW() WHERE id=$1 RETURNING head_sequence`, workspaceID).Scan(&value)
	return value, err
}
func (r *EndgeRepository) CreateMutationBatch(ctx context.Context, workspaceID *string, kind, actor string) (string, error) {
	id := uuid.NewString()
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO mutation_batches(id,workspace_id,kind,actor_user_id) VALUES($1,$2,$3,$4)`, id, workspaceID, kind, actor)
	return id, err
}
func (r *EndgeRepository) LatestRevision(ctx context.Context, workspaceID *string, kind, documentID string) (*entities.Revision, error) {
	row := r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.workspace_id IS NOT DISTINCT FROM $1 AND r.document_type=$2 AND r.document_id=$3 ORDER BY r.revision_number DESC LIMIT 1`, workspaceID, kind, documentID)
	return scanRevision(row)
}
func revisionSelect() string {
	return `SELECT r.id::text,COALESCE(r.workspace_id::text,''),r.document_type,r.document_id::text,r.document_identity,r.revision_number,r.workspace_sequence,r.operation,r.parent_revision_id::text,r.restored_from_revision_id::text,r.committed_in_commit_id::text,r.mutation_batch_id::text,r.snapshot_version,r.snapshot,r.checksum,` + actorScan("u") + `,
		COALESCE((SELECT jsonb_agg(` + actorScan("contributor") + ` ORDER BY contributor.username,contributor.id)
			FROM service_users contributor WHERE contributor.id=ANY(r.contributor_user_ids)),'[]'::jsonb),r.created_at
		FROM document_revisions r JOIN service_users u ON u.id=r.created_by`
}
func scanRevision(row scanner) (*entities.Revision, error) {
	v := &entities.Revision{}
	var workspace string
	var created, contributors []byte
	if err := row.Scan(&v.ID, &workspace, &v.DocumentType, &v.DocumentID, &v.DocumentIdentity, &v.RevisionNumber, &v.WorkspaceSequence, &v.Operation, &v.ParentRevisionID, &v.RestoredFromRevisionID, &v.CommittedInCommitID, &v.MutationBatchID, &v.SnapshotVersion, &v.Snapshot, &v.Checksum, &created, &contributors, &v.CreatedAt); err != nil {
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
func (r *EndgeRepository) ListRevisions(ctx context.Context, workspaceID, kind, identity string) ([]entities.Revision, error) {
	document, err := r.GetDocument(ctx, workspaceID, kind, identity, true)
	if err != nil {
		return nil, err
	}
	rows, err := r.executor(ctx).Query(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 ORDER BY r.revision_number DESC`, workspaceID, kind, document.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Revision{}
	for rows.Next() {
		v, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) GetRevision(ctx context.Context, workspaceID, kind, identity, id string) (*entities.Revision, error) {
	document, err := r.GetDocument(ctx, workspaceID, kind, identity, true)
	if err != nil {
		return nil, err
	}
	return scanRevision(r.executor(ctx).QueryRow(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.document_type=$2 AND r.document_id=$3 AND r.id=$4`, workspaceID, kind, document.ID, id))
}
