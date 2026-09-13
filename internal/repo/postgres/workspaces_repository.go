package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func (r *EndgeRepository) ListWorkspaces(ctx context.Context, userID string, platform bool) ([]entities.Workspace, error) {
	where := "WHERE w.deleted_at IS NULL AND g.user_id=$1"
	if platform {
		where = "WHERE w.deleted_at IS NULL"
	}
	rows, err := r.executor(ctx).Query(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.document_structure,sc.identity,w.configuration,w.meta,w.active,w.deleted_at,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w
		LEFT JOIN access_grants g ON g.workspace_id=w.id AND g.user_id=$1 AND g.scope_type='workspace'
		LEFT JOIN compositions sc ON sc.workspace_id=w.id AND sc.id=w.startup_composition_id
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

// ListDeletedWorkspaces returns authorized Workspace tombstones without
// exposing their domain documents.
func (r *EndgeRepository) ListDeletedWorkspaces(ctx context.Context, userID string, platform bool) ([]entities.Workspace, error) {
	where := "WHERE w.deleted_at IS NOT NULL AND g.user_id=$1"
	if platform {
		where = "WHERE w.deleted_at IS NOT NULL"
	}
	rows, err := r.executor(ctx).Query(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.document_structure,sc.identity,w.configuration,w.meta,w.active,w.deleted_at,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w
		LEFT JOIN access_grants g ON g.workspace_id=w.id AND g.user_id=$1 AND g.scope_type='workspace'
		LEFT JOIN compositions sc ON sc.workspace_id=w.id AND sc.id=w.startup_composition_id
		JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by `+where+` ORDER BY w.deleted_at DESC,w.identity`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Workspace{}
	for rows.Next() {
		value, scanErr := scanWorkspace(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetWorkspace(ctx context.Context, identity string) (*entities.Workspace, error) {
	row := r.executor(ctx).QueryRow(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.document_structure,sc.identity,w.configuration,w.meta,w.active,w.deleted_at,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w LEFT JOIN compositions sc ON sc.workspace_id=w.id AND sc.id=w.startup_composition_id JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by WHERE w.identity=$1 AND w.deleted_at IS NULL`, identity)
	return scanWorkspace(row)
}

func (r *EndgeRepository) GetDeletedWorkspace(ctx context.Context, identity string) (*entities.Workspace, error) {
	row := r.executor(ctx).QueryRow(ctx, `SELECT w.id::text,w.identity,w.display_name,w.description,w.data_mode,w.document_structure,sc.identity,w.configuration,w.meta,w.active,w.deleted_at,w.generation::text,w.head_sequence,w.revision,
		`+actorScan("cu")+`,`+actorScan("uu")+`,w.created_at,w.updated_at FROM workspaces w LEFT JOIN compositions sc ON sc.workspace_id=w.id AND sc.id=w.startup_composition_id JOIN service_users cu ON cu.id=w.created_by JOIN service_users uu ON uu.id=w.updated_by WHERE w.identity=$1 AND w.deleted_at IS NOT NULL`, identity)
	return scanWorkspace(row)
}

type scanner interface{ Scan(...any) error }

func scanWorkspace(row scanner) (*entities.Workspace, error) {
	value := &entities.Workspace{}
	var created, updated []byte
	if err := row.Scan(&value.ID, &value.Identity, &value.DisplayName, &value.Description, &value.DataMode, &value.DocumentStructure, &value.StartupCompositionIdentity, &value.Configuration, &value.Meta, &value.Active, &value.DeletedAt, &value.Generation, &value.HeadSequence, &value.Revision, &created, &updated, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(created, &value.CreatedBy)
	_ = json.Unmarshal(updated, &value.UpdatedBy)
	return value, nil
}

func (r *EndgeRepository) CreateWorkspace(ctx context.Context, value entities.Workspace, actor string) (*entities.Workspace, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspaces(id,identity,display_name,description,data_mode,document_structure,configuration,meta,active,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)`, value.ID, value.Identity, value.DisplayName, value.Description, value.DataMode, value.DocumentStructure, value.Configuration, value.Meta, value.Active, actor)
	if err != nil {
		return nil, err
	}
	return r.GetWorkspace(ctx, value.Identity)
}

// FinalizeWorkspaceBootstrap связывает новый Workspace с созданной в той же транзакции стартовой Composition без искусственного увеличения revision.
func (r *EndgeRepository) FinalizeWorkspaceBootstrap(ctx context.Context, workspaceID, compositionID string) (*entities.Workspace, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces w SET startup_composition_id=$2 FROM compositions c WHERE w.id=$1 AND w.startup_composition_id IS NULL AND c.workspace_id=w.id AND c.id=$2 AND c.active=TRUE AND c.deleted_at IS NULL`, workspaceID, compositionID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("workspace bootstrap conflict")
	}
	return r.getWorkspaceByID(ctx, workspaceID)
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
	if v, ok := patch["documentStructure"].(string); ok {
		current.DocumentStructure = v
	}
	startupChanged := false
	var startupCompositionID *string
	if value, exists := patch["startupCompositionIdentity"]; exists {
		startupChanged = true
		current.StartupCompositionIdentity = nil
		if value != nil {
			startupIdentity, ok := value.(string)
			if !ok || strings.TrimSpace(startupIdentity) == "" {
				return nil, fmt.Errorf("startup composition identity is invalid")
			}
			startupIdentity = strings.TrimSpace(startupIdentity)
			resolved, resolveErr := r.resolveActiveDocumentID(ctx, current.ID, entities.CollectionCompositions, startupIdentity)
			if resolveErr != nil {
				return nil, resolveErr
			}
			startupCompositionID = &resolved
			current.StartupCompositionIdentity = &startupIdentity
		}
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
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces SET identity=$1,display_name=$2,description=$3,data_mode=$4,document_structure=$5,startup_composition_id=CASE WHEN $6 THEN $7::uuid ELSE startup_composition_id END,configuration=$8,meta=$9,active=$10,updated_by=$11,updated_at=NOW(),revision=revision+1 WHERE id=$12 AND revision=$13`, current.Identity, current.DisplayName, current.Description, current.DataMode, current.DocumentStructure, startupChanged, startupCompositionID, current.Configuration, current.Meta, current.Active, actor, current.ID, revision)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetWorkspace(ctx, current.Identity)
}

// SoftDeleteWorkspace помечает Workspace tombstone без удаления его данных и назначений.
func (r *EndgeRepository) SoftDeleteWorkspace(ctx context.Context, workspaceID string, revision int, actor string) (*entities.Workspace, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces SET deleted_at=NOW(),updated_by=$3,updated_at=NOW(),revision=revision+1 WHERE id=$1 AND revision=$2 AND deleted_at IS NULL`, workspaceID, revision, actor)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.getWorkspaceByID(ctx, workspaceID)
}

// RestoreWorkspace clears a Workspace tombstone using optimistic concurrency.
func (r *EndgeRepository) RestoreWorkspace(ctx context.Context, workspaceID string, revision int, actor string) (*entities.Workspace, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces SET deleted_at=NULL,updated_by=$3,updated_at=NOW(),revision=revision+1 WHERE id=$1 AND revision=$2 AND deleted_at IS NOT NULL`, workspaceID, revision, actor)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.getWorkspaceByID(ctx, workspaceID)
}

// ClearStartupComposition атомарно снимает ссылку только если Workspace всё ещё указывает на удаляемую Composition.
func (r *EndgeRepository) ClearStartupComposition(ctx context.Context, workspaceID, compositionID, actor string) (*entities.Workspace, bool, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE workspaces SET startup_composition_id=NULL,updated_by=$3,updated_at=NOW(),revision=revision+1 WHERE id=$1 AND startup_composition_id=$2`, workspaceID, compositionID, actor)
	if err != nil {
		return nil, false, err
	}
	if tag.RowsAffected() == 0 {
		return nil, false, nil
	}
	workspace, err := r.getWorkspaceByID(ctx, workspaceID)
	return workspace, true, err
}

func (r *EndgeRepository) WorkspaceRole(ctx context.Context, workspaceID, userID string, platform bool) (string, error) {
	if platform {
		return "platform_admin", nil
	}
	var role string
	err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(g.role,'') FROM workspaces w LEFT JOIN access_grants g ON g.workspace_id=w.id AND g.user_id=$2 AND g.scope_type='workspace' WHERE w.id=$1 AND w.deleted_at IS NULL`, workspaceID, userID).Scan(&role)
	if err != nil {
		return "", repositoryError(err)
	}
	return role, nil
}

func (r *EndgeRepository) DeletedWorkspaceRole(ctx context.Context, workspaceID, userID string, platform bool) (string, error) {
	if platform {
		return "platform_admin", nil
	}
	var role string
	err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(g.role,'') FROM workspaces w LEFT JOIN access_grants g ON g.workspace_id=w.id AND g.user_id=$2 AND g.scope_type='workspace' WHERE w.id=$1 AND w.deleted_at IS NOT NULL`, workspaceID, userID).Scan(&role)
	if err != nil {
		return "", repositoryError(err)
	}
	return role, nil
}

func (r *EndgeRepository) ListMemberships(ctx context.Context, workspaceID string) ([]entities.Membership, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT g.workspace_id::text,g.user_id::text,g.role,u.username,u.display_name,g.created_at,g.updated_at FROM access_grants g JOIN service_users u ON u.id=g.user_id WHERE g.scope_type='workspace' AND g.workspace_id=$1 ORDER BY u.username,u.id`, workspaceID)
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
	if err := r.invalidateExternalAccess(ctx, userID); err != nil {
		return nil, err
	}
	tag, err := r.executor(ctx).Exec(ctx, `INSERT INTO access_grants(user_id,scope_type,workspace_id,role,created_by,updated_by)
		SELECT u.id,'workspace',$1,$3,$4,$4 FROM service_users u JOIN workspaces w ON w.id=$1
		WHERE u.id=$2 AND u.active=TRUE AND u.is_system=FALSE ON CONFLICT(workspace_id,user_id) WHERE scope_type='workspace'
		DO UPDATE SET role=EXCLUDED.role,updated_by=EXCLUDED.updated_by,updated_at=NOW()`, workspaceID, userID, role, actor)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}
	_, err = r.executor(ctx).Exec(ctx, `INSERT INTO workspace_memberships(workspace_id,user_id,role,created_by) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role,updated_at=NOW()`, workspaceID, userID, role, actor)
	if err != nil {
		return nil, err
	}
	var v entities.Membership
	err = r.executor(ctx).QueryRow(ctx, `SELECT g.workspace_id::text,g.user_id::text,g.role,u.username,u.display_name,g.created_at,g.updated_at FROM access_grants g JOIN service_users u ON u.id=g.user_id WHERE g.scope_type='workspace' AND g.workspace_id=$1 AND g.user_id=$2`, workspaceID, userID).Scan(&v.WorkspaceID, &v.UserID, &v.Role, &v.Username, &v.DisplayName, &v.CreatedAt, &v.UpdatedAt)
	return &v, repositoryError(err)
}
func (r *EndgeRepository) DeleteMembership(ctx context.Context, workspaceID, userID string) error {
	if err := r.invalidateExternalAccess(ctx, userID); err != nil {
		return err
	}
	if _, err := r.executor(ctx).Exec(ctx, `DELETE FROM workspace_memberships WHERE workspace_id=$1 AND user_id=$2`, workspaceID, userID); err != nil {
		return err
	}
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM access_grants WHERE scope_type='workspace' AND workspace_id=$1 AND user_id=$2`, workspaceID, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
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
