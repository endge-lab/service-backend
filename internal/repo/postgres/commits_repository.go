package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func commitSelect() string {
	return `SELECT c.id::text,c.workspace_id::text,c.parent_commit_id::text,c.base_sequence,c.head_sequence,c.message,c.revision_policy,c.operation,COALESCE(c.domain_version,''),` + actorScan("u") + `,c.created_at FROM workspace_commits c JOIN service_users u ON u.id=c.created_by`
}
func scanCommit(row scanner) (*entities.Commit, error) {
	v := &entities.Commit{}
	var actor []byte
	if err := row.Scan(&v.ID, &v.WorkspaceID, &v.ParentCommitID, &v.BaseSequence, &v.HeadSequence, &v.Message, &v.RevisionPolicy, &v.Operation, &v.DomainVersion, &actor, &v.CreatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(actor, &v.CreatedBy)
	return v, nil
}
func (r *EndgeRepository) LatestCommit(ctx context.Context, workspaceID string) (*entities.Commit, error) {
	return scanCommit(r.executor(ctx).QueryRow(ctx, commitSelect()+` WHERE c.workspace_id=$1 ORDER BY c.head_sequence DESC LIMIT 1`, workspaceID))
}
func (r *EndgeRepository) ListCommits(ctx context.Context, workspaceID string) ([]entities.Commit, error) {
	rows, err := r.executor(ctx).Query(ctx, commitSelect()+` WHERE c.workspace_id=$1 ORDER BY c.created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Commit{}
	for rows.Next() {
		v, err := scanCommit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) GetCommit(ctx context.Context, workspaceID, id string) (*entities.Commit, error) {
	v, err := scanCommit(r.executor(ctx).QueryRow(ctx, commitSelect()+` WHERE c.workspace_id=$1 AND c.id=$2`, workspaceID, id))
	if err != nil {
		return nil, err
	}
	rows, err := r.executor(ctx).Query(ctx, `SELECT c.document_type,c.document_id::text,
		COALESCE(after_revision.document_identity,before_revision.document_identity,''),
		c.before_revision_id::text,c.after_revision_id::text,c.operation
		FROM workspace_commit_changes c
		LEFT JOIN document_revisions before_revision ON before_revision.id=c.before_revision_id
		LEFT JOIN document_revisions after_revision ON after_revision.id=c.after_revision_id
		WHERE c.commit_id=$1
		ORDER BY c.document_type,COALESCE(after_revision.document_identity,before_revision.document_identity,''),c.document_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c entities.CommitChange
		if err := rows.Scan(&c.DocumentType, &c.DocumentID, &c.DocumentIdentity, &c.BeforeRevisionID, &c.AfterRevisionID, &c.Operation); err != nil {
			return nil, err
		}
		v.Changes = append(v.Changes, c)
	}
	return v, rows.Err()
}
func (r *EndgeRepository) PendingRevisions(ctx context.Context, workspaceID string, base int64) ([]entities.Revision, error) {
	rows, err := r.executor(ctx).Query(ctx, revisionSelect()+` WHERE r.workspace_id=$1 AND r.workspace_sequence>$2 AND r.committed_in_commit_id IS NULL ORDER BY r.workspace_sequence`, workspaceID, base)
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
func (r *EndgeRepository) CreateCommit(ctx context.Context, v entities.Commit, changes []entities.CommitChange) (*entities.Commit, error) {
	bundle, err := r.ExportWorkspace(ctx, v.WorkspaceID, &v.HeadSequence)
	if err != nil {
		return nil, err
	}
	v.DomainVersion, err = domainversion.ComputeRaw(bundle)
	if err != nil {
		return nil, err
	}
	_, err = r.executor(ctx).Exec(ctx, `INSERT INTO workspace_commits(id,workspace_id,parent_commit_id,base_sequence,head_sequence,message,revision_policy,operation,domain_version,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, v.ID, v.WorkspaceID, v.ParentCommitID, v.BaseSequence, v.HeadSequence, v.Message, v.RevisionPolicy, v.Operation, v.DomainVersion, v.CreatedBy.ID)
	if err != nil {
		return nil, err
	}
	if _, err = r.executor(ctx).Exec(ctx, `INSERT INTO workspace_commit_integrations(commit_id,integration_identity,version,configuration) SELECT $1,i.identity,wi.version,wi.configuration FROM workspace_integrations wi JOIN integrations i ON i.id=wi.integration_id WHERE wi.workspace_id=$2`, v.ID, v.WorkspaceID); err != nil {
		return nil, err
	}
	for _, c := range changes {
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO workspace_commit_changes(commit_id,document_type,document_id,before_revision_id,after_revision_id,operation) VALUES($1,$2,$3,$4,$5,$6)`, v.ID, c.DocumentType, c.DocumentID, c.BeforeRevisionID, c.AfterRevisionID, c.Operation)
		if err != nil {
			return nil, err
		}
	}
	return r.GetCommit(ctx, v.WorkspaceID, v.ID)
}
func (r *EndgeRepository) AttachRevisionsToCommit(ctx context.Context, commitID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.executor(ctx).Exec(ctx, `UPDATE document_revisions SET committed_in_commit_id=$1 WHERE id=ANY($2::uuid[])`, commitID, ids)
	return err
}
func (r *EndgeRepository) SquashPending(ctx context.Context, workspaceID string, base int64, actor, batch string) ([]entities.Revision, error) {
	pending, err := r.PendingRevisions(ctx, workspaceID, base)
	if err != nil {
		return nil, err
	}
	groups := map[string][]entities.Revision{}
	for _, v := range pending {
		groups[v.DocumentType+":"+v.DocumentID] = append(groups[v.DocumentType+":"+v.DocumentID], v)
	}
	result := []entities.Revision{}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items := groups[key]
		last := items[len(items)-1]
		var parent *string
		err := r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM document_revisions WHERE workspace_id=$1 AND document_type=$2 AND document_id=$3 AND workspace_sequence<=$4 AND committed_in_commit_id IS NOT NULL ORDER BY workspace_sequence DESC LIMIT 1`, workspaceID, last.DocumentType, last.DocumentID, base).Scan(&parent)
		if errors.Is(err, pgx.ErrNoRows) {
			parent = nil
		} else if err != nil {
			return nil, err
		}
		ids := []string{}
		contributors := map[string]bool{}
		for _, item := range items {
			ids = append(ids, item.ID)
			contributors[item.CreatedBy.ID] = true
			for _, contributorID := range item.ContributorUserIDs {
				contributors[contributorID] = true
			}
		}
		_, err = r.executor(ctx).Exec(ctx, `DELETE FROM document_revisions WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return nil, err
		}
		last.ID = uuid.NewString()
		last.Operation = "squash"
		last.ParentRevisionID = parent
		last.MutationBatchID = batch
		last.CreatedBy.ID = actor
		last.ContributorUserIDs = make([]string, 0, len(contributors))
		for userID := range contributors {
			last.ContributorUserIDs = append(last.ContributorUserIDs, userID)
		}
		sort.Strings(last.ContributorUserIDs)
		created, err := r.InsertRevision(ctx, last)
		if err != nil {
			return nil, err
		}
		result = append(result, *created)
	}
	return result, nil
}
