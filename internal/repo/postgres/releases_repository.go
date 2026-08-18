package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/jackc/pgx/v5"
)

func releaseMetadataSelect() string {
	return `SELECT r.id::text,r.workspace_id::text,r.identity,r.display_name,r.description,r.source_commit_id::text,r.head_sequence,r.schema_version,r.checksum,` + actorScan("u") + `,r.created_at FROM releases r JOIN service_users u ON u.id=r.created_by`
}
func scanRelease(row scanner) (*entities.Release, error) {
	v := &entities.Release{}
	var actor []byte
	if err := row.Scan(&v.ID, &v.WorkspaceID, &v.Identity, &v.DisplayName, &v.Description, &v.SourceCommitID, &v.HeadSequence, &v.SchemaVersion, &v.Checksum, &actor, &v.CreatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(actor, &v.CreatedBy)
	return v, nil
}
func (r *EndgeRepository) CreateRelease(ctx context.Context, v entities.Release, artifact json.RawMessage) (*entities.Release, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO releases(id,workspace_id,identity,display_name,description,source_commit_id,head_sequence,schema_version,checksum,data,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, v.SourceCommitID, v.HeadSequence, v.SchemaVersion, v.Checksum, artifact, v.CreatedBy.ID)
	if err != nil {
		return nil, err
	}
	return r.GetReleaseMetadata(ctx, v.WorkspaceID, v.Identity)
}
func (r *EndgeRepository) ListReleases(ctx context.Context, workspaceID string) ([]entities.Release, error) {
	rows, err := r.executor(ctx).Query(ctx, releaseMetadataSelect()+` WHERE r.workspace_id=$1 ORDER BY r.created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Release{}
	for rows.Next() {
		v, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) GetReleaseMetadata(ctx context.Context, workspaceID, identity string) (*entities.Release, error) {
	return scanRelease(r.executor(ctx).QueryRow(ctx, releaseMetadataSelect()+` WHERE r.workspace_id=$1 AND r.identity=$2`, workspaceID, identity))
}

func (r *EndgeRepository) GetLatestReleaseMetadata(ctx context.Context, workspaceID string) (*entities.Release, error) {
	return scanRelease(r.executor(ctx).QueryRow(ctx, releaseMetadataSelect()+` WHERE r.workspace_id=$1 ORDER BY r.created_at DESC,r.id DESC LIMIT 1`, workspaceID))
}

func (r *EndgeRepository) GetReleaseArtifact(ctx context.Context, workspaceID, releaseID string) (*entities.ReleaseArtifact, error) {
	value := &entities.ReleaseArtifact{}
	err := r.executor(ctx).QueryRow(ctx, `SELECT id::text,workspace_id::text,identity,checksum,data FROM releases WHERE workspace_id=$1 AND id=$2`, workspaceID, releaseID).Scan(
		&value.ReleaseID, &value.WorkspaceID, &value.Identity, &value.Checksum, &value.Data)
	if err != nil {
		return nil, repositoryError(err)
	}
	return value, nil
}

func (r *EndgeRepository) ExportWorkspace(ctx context.Context, workspaceID string, head *int64) (json.RawMessage, error) {
	workspace, err := r.getWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if head != nil {
		var raw []byte
		revisionErr := r.executor(ctx).QueryRow(ctx, `SELECT snapshot FROM document_revisions WHERE workspace_id=$1 AND document_type='workspaces' AND workspace_sequence<=$2 ORDER BY workspace_sequence DESC LIMIT 1`, workspaceID, *head).Scan(&raw)
		if revisionErr == nil {
			var historical entities.Workspace
			if err = json.Unmarshal(raw, &historical); err != nil {
				return nil, err
			}
			workspace = &historical
		} else if !errors.Is(revisionErr, pgx.ErrNoRows) {
			return nil, revisionErr
		}
	}
	bundle := entities.PortableBundle{Kind: "workspace-snapshot", SchemaVersion: 1, Workspace: map[string]any{"identity": workspace.Identity, "displayName": workspace.DisplayName, "description": workspace.Description, "dataMode": workspace.DataMode, "configuration": json.RawMessage(workspace.Configuration), "meta": json.RawMessage(workspace.Meta), "active": workspace.Active}, Documents: map[string][]map[string]any{}, InstalledIntegrations: []map[string]any{}}
	documents := map[string][]entities.Document{}
	if head == nil {
		for kind := range documentTables {
			docs, err := r.listAllActiveDocuments(ctx, workspaceID, kind)
			if err != nil {
				return nil, err
			}
			documents[kind] = docs
		}
	} else {
		rows, err := r.executor(ctx).Query(ctx, `SELECT DISTINCT ON (document_type,document_id) snapshot FROM document_revisions WHERE workspace_id=$1 AND workspace_sequence<=$2 ORDER BY document_type,document_id,workspace_sequence DESC`, workspaceID, *head)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return nil, err
			}
			var doc entities.Document
			if err = json.Unmarshal(raw, &doc); err != nil {
				return nil, err
			}
			if doc.DeletedAt == nil {
				documents[doc.Type] = append(documents[doc.Type], doc)
			}
		}
		if err = rows.Err(); err != nil {
			return nil, err
		}
	}
	for kind := range documentTables {
		items := []map[string]any{}
		for _, doc := range documents[kind] {
			var data map[string]any
			_ = json.Unmarshal(doc.Data, &data)
			if kind == "folders" && doc.ManagedBy == "system" {
				continue
			}
			item := map[string]any{"identity": doc.Identity, "displayName": doc.DisplayName, "description": doc.Description, "folderIdentity": doc.FolderIdentity, "managedBy": doc.ManagedBy, "managedById": doc.ManagedByID, "meta": json.RawMessage(doc.Meta), "active": doc.Active}
			for key, value := range data {
				item[key] = value
			}
			items = append(items, item)
		}
		bundle.Documents[kind] = items
	}
	integrationQuery := `SELECT i.identity,wi.version,wi.configuration FROM workspace_integrations wi JOIN integrations i ON i.id=wi.integration_id WHERE wi.workspace_id=$1 ORDER BY i.identity`
	integrationArgs := []any{workspaceID}
	if head != nil {
		integrationQuery = `SELECT ci.integration_identity,ci.version,ci.configuration FROM workspace_commit_integrations ci JOIN workspace_commits c ON c.id=ci.commit_id WHERE c.workspace_id=$1 AND c.head_sequence=$2 ORDER BY ci.integration_identity`
		integrationArgs = append(integrationArgs, *head)
	}
	rows, err := r.executor(ctx).Query(ctx, integrationQuery, integrationArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var identity, version string
		var configuration json.RawMessage
		if err = rows.Scan(&identity, &version, &configuration); err != nil {
			return nil, err
		}
		bundle.InstalledIntegrations = append(bundle.InstalledIntegrations, map[string]any{"identity": identity, "version": version, "configuration": configuration})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = domainversion.Attach(&bundle); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(bundle)
	return raw, err
}
func (r *EndgeRepository) getWorkspaceByID(ctx context.Context, id string) (*entities.Workspace, error) {
	row := r.executor(ctx).QueryRow(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.configuration,w.meta,w.active,w.generation::text,w.head_sequence,w.revision,`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by WHERE w.id=$1`, id)
	return scanWorkspace(row)
}
