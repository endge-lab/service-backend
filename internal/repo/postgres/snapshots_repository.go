package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// ExportLiveWorkspace возвращает рабочее состояние workspace с локальными server state-полями.
func (r *EndgeRepository) ExportLiveWorkspace(ctx context.Context, workspaceID string) (json.RawMessage, error) {
	workspace, err := r.getWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	bundle := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{
			"identity": workspace.Identity, "displayName": workspace.DisplayName, "description": workspace.Description,
			"dataMode": workspace.DataMode, "configuration": json.RawMessage(workspace.Configuration), "meta": json.RawMessage(workspace.Meta),
			"active": workspace.Active,
			"state":  map[string]any{"id": workspace.ID, "generation": workspace.Generation, "headSequence": workspace.HeadSequence, "revision": workspace.Revision, "createdBy": workspace.CreatedBy, "updatedBy": workspace.UpdatedBy, "createdAt": workspace.CreatedAt, "updatedAt": workspace.UpdatedAt},
		},
		Documents: map[string][]map[string]any{}, InstalledIntegrations: []map[string]any{},
	}
	kinds := make([]string, 0, len(documentTables))
	for kind := range documentTables {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		documents, listErr := r.listAllDocuments(ctx, workspaceID, kind, true)
		if listErr != nil {
			return nil, listErr
		}
		items := make([]map[string]any, 0, len(documents))
		for _, document := range documents {
			var item map[string]any
			_ = json.Unmarshal(document.Data, &item)
			item["identity"] = document.Identity
			item["displayName"] = document.DisplayName
			item["description"] = document.Description
			item["folderIdentity"] = document.FolderIdentity
			item["managedBy"] = document.ManagedBy
			item["managedById"] = document.ManagedByID
			item["meta"] = json.RawMessage(document.Meta)
			item["active"] = document.Active
			item["state"] = map[string]any{"id": document.ID, "revision": document.Revision, "deletedAt": document.DeletedAt, "createdBy": document.CreatedBy, "updatedBy": document.UpdatedBy, "createdAt": document.CreatedAt, "updatedAt": document.UpdatedAt}
			items = append(items, item)
		}
		bundle.Documents[kind] = items
	}
	bindings, err := r.ListWorkspaceIntegrations(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	bundle.InstalledIntegrations = bindings
	return json.Marshal(bundle)
}

// LockWorkspaceSnapshot блокирует строку workspace до завершения текущей транзакции.
// Все document mutations обновляют эту строку при выдаче workspace sequence, поэтому
// после получения блокировки backup/import видит согласованное состояние домена.
func (r *EndgeRepository) LockWorkspaceSnapshot(ctx context.Context, workspaceID string) error {
	var id string
	err := r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM workspaces WHERE id=$1 FOR UPDATE`, workspaceID).Scan(&id)
	return repositoryError(err)
}

// CreateSnapshotImportPlan сохраняет проверенный snapshot до подтверждения импорта.
func (r *EndgeRepository) CreateSnapshotImportPlan(ctx context.Context, value entities.SnapshotImportPlan) (*entities.SnapshotImportPlan, error) {
	_, _ = r.executor(ctx).Exec(ctx, `DELETE FROM workspace_snapshot_import_plans WHERE expires_at<=NOW() OR applied_at IS NOT NULL`)
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspace_snapshot_import_plans(id,workspace_id,snapshot_checksum,snapshot,expected_generation,expected_head_sequence,created_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, value.WorkspaceID, value.SnapshotChecksum, value.Snapshot, value.ExpectedGeneration, value.ExpectedHeadSequence, value.CreatedBy.ID, value.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return r.GetSnapshotImportPlan(ctx, value.WorkspaceID, value.ID, value.CreatedBy.ID)
}

// GetSnapshotImportPlan возвращает принадлежащий actor незавершённый план.
func (r *EndgeRepository) GetSnapshotImportPlan(ctx context.Context, workspaceID, id, actorID string) (*entities.SnapshotImportPlan, error) {
	value := &entities.SnapshotImportPlan{}
	var actor []byte
	err := r.executor(ctx).QueryRow(ctx, `SELECT p.id::text,p.workspace_id::text,p.snapshot_checksum,p.snapshot,p.expected_generation::text,p.expected_head_sequence,`+actorScan("u")+`,p.created_at,p.expires_at,p.applied_at FROM workspace_snapshot_import_plans p JOIN service_users u ON u.id=p.created_by WHERE p.workspace_id=$1 AND p.id=$2 AND p.created_by=$3`, workspaceID, id, actorID).Scan(&value.ID, &value.WorkspaceID, &value.SnapshotChecksum, &value.Snapshot, &value.ExpectedGeneration, &value.ExpectedHeadSequence, &actor, &value.CreatedAt, &value.ExpiresAt, &value.AppliedAt)
	if err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(actor, &value.CreatedBy)
	return value, nil
}

// MarkSnapshotImportPlanApplied помечает план использованным.
func (r *EndgeRepository) MarkSnapshotImportPlanApplied(ctx context.Context, id string) error {
	_, err := r.executor(ctx).Exec(ctx, `UPDATE workspace_snapshot_import_plans SET applied_at=NOW() WHERE id=$1`, id)
	return err
}

// CreateSnapshotBackup сохраняет переносимый backup workspace.
func (r *EndgeRepository) CreateSnapshotBackup(ctx context.Context, value entities.SnapshotBackup) (*entities.SnapshotBackup, error) {
	_, _ = r.executor(ctx).Exec(ctx, `DELETE FROM workspace_snapshot_backups WHERE expires_at IS NOT NULL AND expires_at<=NOW()`)
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspace_snapshot_backups(id,workspace_id,kind,description,schema_version,checksum,data,created_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, value.ID, value.WorkspaceID, value.Kind, value.Description, value.SchemaVersion, value.Checksum, value.Data, value.CreatedBy.ID, value.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return r.GetSnapshotBackup(ctx, value.WorkspaceID, value.ID, true)
}

func snapshotBackupSelect(includeData bool) string {
	data := "NULL::jsonb"
	if includeData {
		data = "b.data"
	}
	return `SELECT b.id::text,b.workspace_id::text,b.kind,b.description,b.schema_version,b.checksum,octet_length(b.data::text),` + data + `,` + actorScan("u") + `,b.created_at,b.expires_at FROM workspace_snapshot_backups b JOIN service_users u ON u.id=b.created_by`
}

func scanSnapshotBackup(row scanner) (*entities.SnapshotBackup, error) {
	value := &entities.SnapshotBackup{}
	var actor []byte
	if err := row.Scan(&value.ID, &value.WorkspaceID, &value.Kind, &value.Description, &value.SchemaVersion, &value.Checksum, &value.SizeBytes, &value.Data, &actor, &value.CreatedAt, &value.ExpiresAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(actor, &value.CreatedBy)
	return value, nil
}

// ListSnapshotBackups возвращает backups текущего workspace.
func (r *EndgeRepository) ListSnapshotBackups(ctx context.Context, workspaceID, kind string, includeData bool, limit, offset int) ([]entities.SnapshotBackup, error) {
	_, _ = r.executor(ctx).Exec(ctx, `DELETE FROM workspace_snapshot_backups WHERE expires_at IS NOT NULL AND expires_at<=NOW()`)
	query := snapshotBackupSelect(includeData) + ` WHERE b.workspace_id=$1 AND (b.expires_at IS NULL OR b.expires_at>NOW())`
	args := []any{workspaceID}
	if kind != "" {
		query += ` AND b.kind=$2`
		args = append(args, kind)
	}
	query += ` ORDER BY b.created_at DESC,b.id DESC`
	if limit > 0 {
		args = append(args, limit, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := r.executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.SnapshotBackup{}
	for rows.Next() {
		value, scanErr := scanSnapshotBackup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

// GetSnapshotBackup возвращает backup по UUID или read-only alias last.
func (r *EndgeRepository) GetSnapshotBackup(ctx context.Context, workspaceID, id string, includeData bool) (*entities.SnapshotBackup, error) {
	query := snapshotBackupSelect(includeData) + ` WHERE b.workspace_id=$1 AND (b.expires_at IS NULL OR b.expires_at>NOW())`
	args := []any{workspaceID}
	if id == "last" {
		query += ` ORDER BY b.created_at DESC,b.id DESC LIMIT 1`
	} else {
		query += ` AND b.id=$2`
		args = append(args, id)
	}
	return scanSnapshotBackup(r.executor(ctx).QueryRow(ctx, query, args...))
}
