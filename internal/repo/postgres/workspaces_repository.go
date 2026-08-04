package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func (r *EndgeRepository) ListWorkspaces(ctx context.Context, userID string, platform bool) ([]entities.Workspace, error) {
	where := "WHERE w.identity='default' OR m.user_id=$1"
	if platform {
		where = "WHERE TRUE"
	}
	rows, err := r.executor(ctx).Query(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.configuration,w.meta,w.active,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w
		LEFT JOIN workspace_memberships m ON m.workspace_id=w.id AND m.user_id=$1
		JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by `+where+` ORDER BY w.identity`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Workspace{}
	for rows.Next() {
		value, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetWorkspace(ctx context.Context, identity string) (*entities.Workspace, error) {
	row := r.executor(ctx).QueryRow(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.configuration,w.meta,w.active,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by WHERE w.identity=$1`, identity)
	return scanWorkspace(row)
}

type scanner interface{ Scan(...any) error }

func scanWorkspace(row scanner) (*entities.Workspace, error) {
	value := &entities.Workspace{}
	var created, updated []byte
	if err := row.Scan(&value.ID, &value.Identity, &value.DisplayName, &value.Description, &value.DataMode, &value.Configuration, &value.Meta, &value.Active, &value.Generation, &value.HeadSequence, &value.Revision, &created, &updated, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(created, &value.CreatedBy)
	_ = json.Unmarshal(updated, &value.UpdatedBy)
	return value, nil
}

func (r *EndgeRepository) CreateWorkspace(ctx context.Context, value entities.Workspace, actor string) (*entities.Workspace, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspaces(id,identity,display_name,description,data_mode,configuration,meta,active,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, value.ID, value.Identity, value.DisplayName, value.Description, value.DataMode, value.Configuration, value.Meta, value.Active, actor)
	if err != nil {
		return nil, err
	}
	return r.GetWorkspace(ctx, value.Identity)
}

func (r *EndgeRepository) UpdateWorkspace(ctx context.Context, identity string, patch map[string]any, revision int, actor string) (*entities.Workspace, error) {
	current, err := r.GetWorkspace(ctx, identity)
	if err != nil {
		return nil, err
	}
	if v, ok := patch["identity"].(string); ok {
		current.Identity = strings.TrimSpace(v)
	}
	if v, ok := patch["displayName"].(string); ok {
		current.DisplayName = v
	}
	if _, ok := patch["description"]; ok {
		if v, isString := patch["description"].(string); isString {
			current.Description = &v
		} else {
			current.Description = nil
		}
	}
	if v, ok := patch["dataMode"].(string); ok {
		current.DataMode = v
	}
	if v, ok := patch["configuration"]; ok {
		current.Configuration = mustJSON(v)
	}
	if v, ok := patch["meta"]; ok {
		current.Meta = mustJSON(v)
	}
	if v, ok := patch["active"].(bool); ok {
		current.Active = v
	}
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces SET identity=$1,display_name=$2,description=$3,data_mode=$4,configuration=$5,meta=$6,active=$7,updated_by=$8,updated_at=NOW(),revision=revision+1 WHERE id=$9 AND revision=$10`, current.Identity, current.DisplayName, current.Description, current.DataMode, current.Configuration, current.Meta, current.Active, actor, current.ID, revision)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetWorkspace(ctx, current.Identity)
}

func (r *EndgeRepository) WorkspaceRole(ctx context.Context, workspaceID, userID string, platform bool) (string, error) {
	if platform {
		return "platform_admin", nil
	}
	var identity, role string
	err := r.executor(ctx).QueryRow(ctx, `SELECT w.identity,COALESCE(m.role,'') FROM workspaces w LEFT JOIN workspace_memberships m ON m.workspace_id=w.id AND m.user_id=$2 WHERE w.id=$1`, workspaceID, userID).Scan(&identity, &role)
	if err != nil {
		return "", repositoryError(err)
	}
	if role != "" {
		return role, nil
	}
	if identity == "default" {
		return "editor", nil
	}
	return "", nil
}

func (r *EndgeRepository) ListMemberships(ctx context.Context, workspaceID string) ([]entities.Membership, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT m.workspace_id::text,m.user_id::text,m.role,u.username,u.display_name,m.created_at,m.updated_at FROM workspace_memberships m JOIN service_users u ON u.id=m.user_id WHERE m.workspace_id=$1 ORDER BY u.username,u.id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Membership{}
	for rows.Next() {
		var v entities.Membership
		if err := rows.Scan(&v.WorkspaceID, &v.UserID, &v.Role, &v.Username, &v.DisplayName, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) PutMembership(ctx context.Context, workspaceID, userID, role, actor string) (*entities.Membership, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspace_memberships(workspace_id,user_id,role,created_by) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role,updated_at=NOW()`, workspaceID, userID, role, actor)
	if err != nil {
		return nil, err
	}
	var v entities.Membership
	err = r.executor(ctx).QueryRow(ctx, `SELECT m.workspace_id::text,m.user_id::text,m.role,u.username,u.display_name,m.created_at,m.updated_at FROM workspace_memberships m JOIN service_users u ON u.id=m.user_id WHERE m.workspace_id=$1 AND m.user_id=$2`, workspaceID, userID).Scan(&v.WorkspaceID, &v.UserID, &v.Role, &v.Username, &v.DisplayName, &v.CreatedAt, &v.UpdatedAt)
	return &v, repositoryError(err)
}
func (r *EndgeRepository) DeleteMembership(ctx context.Context, workspaceID, userID string) error {
	_, err := r.executor(ctx).Exec(ctx, `DELETE FROM workspace_memberships WHERE workspace_id=$1 AND user_id=$2`, workspaceID, userID)
	return err
}
func (r *EndgeRepository) ReplaceWorkspaceIntegrations(ctx context.Context, workspaceID string, items []map[string]any, actor string) error {
	if _, err := r.executor(ctx).Exec(ctx, `DELETE FROM workspace_integrations WHERE workspace_id=$1`, workspaceID); err != nil {
		return err
	}
	for _, item := range items {
		identity := stringValue(item["identity"])
		version := stringValue(item["version"])
		if identity == "" || version == "" {
			return fmt.Errorf("integration identity and version are required")
		}
		configuration := mustJSON(item["configuration"])
		tag, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspace_integrations(workspace_id,integration_id,version,configuration,created_by) SELECT $1,id,$3,$4,$5 FROM integrations WHERE identity=$2 AND deleted_at IS NULL`, workspaceID, identity, version, configuration, actor)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("integration %s not found", identity)
		}
	}
	return nil
}

func (r *EndgeRepository) ListWorkspaceIntegrations(ctx context.Context, workspaceID string) ([]map[string]any, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT i.identity,wi.version,wi.configuration FROM workspace_integrations wi JOIN integrations i ON i.id=wi.integration_id WHERE wi.workspace_id=$1 ORDER BY i.identity`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []map[string]any{}
	for rows.Next() {
		var identity, version string
		var configuration json.RawMessage
		if err = rows.Scan(&identity, &version, &configuration); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"identity": identity, "version": version, "configuration": configuration})
	}
	return result, rows.Err()
}
