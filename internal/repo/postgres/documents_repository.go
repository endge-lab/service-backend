package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func tableFor(kind string) (string, error) {
	table, ok := documentTables[kind]
	if !ok {
		return "", fmt.Errorf("unsupported document type %q", kind)
	}
	return table, nil
}

func (r *EndgeRepository) ListDocuments(ctx context.Context, workspaceID, kind string, filter ports.DocumentFilter) ([]entities.Document, error) {
	table, err := tableFor(kind)
	if err != nil {
		return nil, err
	}
	args := []any{workspaceID}
	where := []string{"d.workspace_id=$1"}
	if !filter.IncludeDeleted {
		where = append(where, "d.deleted_at IS NULL")
	}
	if filter.FolderIdentity != nil {
		args = append(args, *filter.FolderIdentity)
		where = append(where, fmt.Sprintf("f.identity=$%d", len(args)))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		where = append(where, fmt.Sprintf("d.active=$%d", len(args)))
	}
	args = append(args, filter.Limit, filter.Offset)
	query := documentSelect(table, kind) + ` WHERE ` + strings.Join(where, " AND ") + fmt.Sprintf(" ORDER BY d.identity LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Document{}
	for rows.Next() {
		v, err := scanDocument(rows, kind)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}

// listAllActiveDocuments читает полный набор документов для snapshot/export без
// пользовательской пагинации и искусственного ограничения размера коллекции.
func (r *EndgeRepository) listAllActiveDocuments(ctx context.Context, workspaceID, kind string) ([]entities.Document, error) {
	return r.listAllDocuments(ctx, workspaceID, kind, false)
}

// listAllDocuments читает полный набор документов без пользовательской пагинации.
func (r *EndgeRepository) listAllDocuments(ctx context.Context, workspaceID, kind string, includeDeleted bool) ([]entities.Document, error) {
	table, err := tableFor(kind)
	if err != nil {
		return nil, err
	}
	where := ` WHERE d.workspace_id=$1`
	if !includeDeleted {
		where += ` AND d.deleted_at IS NULL`
	}
	rows, err := r.executor(ctx).Query(ctx, documentSelect(table, kind)+where+` ORDER BY d.identity`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Document{}
	for rows.Next() {
		value, scanErr := scanDocument(rows, kind)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) GetDocument(ctx context.Context, workspaceID, kind, identity string, includeDeleted bool) (*entities.Document, error) {
	table, err := tableFor(kind)
	if err != nil {
		return nil, err
	}
	query := documentSelect(table, kind) + ` WHERE d.workspace_id=$1 AND d.identity=$2`
	if !includeDeleted {
		query += ` AND d.deleted_at IS NULL`
	}
	return scanDocument(r.executor(ctx).QueryRow(ctx, query, workspaceID, identity), kind)
}

func documentSelect(table, kind string) string {
	data := "d.data"
	folderJoin := "LEFT JOIN folders f ON f.id=d.folder_id AND f.workspace_id=d.workspace_id"
	if kind == "folders" {
		data = `jsonb_build_object('entityType',d.entity_type,'parentIdentity',pf.identity,'isRoot',d.is_root)`
		folderJoin = "LEFT JOIN folders f ON f.id=d.parent_id AND f.workspace_id=d.workspace_id LEFT JOIN folders pf ON pf.id=d.parent_id AND pf.workspace_id=d.workspace_id"
	} else if kind == "updates" {
		data = `(d.data - 'storeIdentity') || jsonb_build_object('storeIdentity',store.identity)`
		folderJoin += " JOIN stores store ON store.id=d.store_id AND store.workspace_id=d.workspace_id"
	} else if kind == "vocabs" {
		data = `(d.data - 'authProfileIdentity') || jsonb_build_object('authProfileIdentity',auth_profile.identity)`
		folderJoin += " LEFT JOIN auth_profiles auth_profile ON auth_profile.id=d.auth_profile_id AND auth_profile.workspace_id=d.workspace_id"
	} else if kind == "projects" {
		data = `(d.data - 'navigationIdentity') || jsonb_build_object('navigationIdentity',navigation.identity)`
		folderJoin += " LEFT JOIN navigations navigation ON navigation.id=d.navigation_id AND navigation.workspace_id=d.workspace_id"
	}
	return `SELECT d.id::text,d.workspace_id::text,d.identity,d.display_name,d.description,f.identity,d.managed_by,d.managed_by_id,d.meta,` + data + `,d.active,d.deleted_at,d.revision,` + actorScan("cu") + `,` + actorScan("uu") + `,d.created_at,d.updated_at FROM ` + table + ` d ` + folderJoin + ` JOIN service_users cu ON cu.id=d.created_by JOIN service_users uu ON uu.id=d.updated_by`
}
func scanDocument(row scanner, kind string) (*entities.Document, error) {
	v := &entities.Document{Type: kind}
	var created, updated []byte
	if err := row.Scan(&v.ID, &v.WorkspaceID, &v.Identity, &v.DisplayName, &v.Description, &v.FolderIdentity, &v.ManagedBy, &v.ManagedByID, &v.Meta, &v.Data, &v.Active, &v.DeletedAt, &v.Revision, &created, &updated, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(created, &v.CreatedBy)
	_ = json.Unmarshal(updated, &v.UpdatedBy)
	return v, nil
}

func (r *EndgeRepository) InsertDocument(ctx context.Context, v entities.Document, folderID *string) (*entities.Document, error) {
	table, err := tableFor(v.Type)
	if err != nil {
		return nil, err
	}
	if v.Type == "folders" {
		var data map[string]any
		_ = json.Unmarshal(v.Data, &data)
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO folders(id,workspace_id,identity,display_name,description,entity_type,parent_id,is_root,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, stringValue(data["entityType"]), folderID, boolValue(data["isRoot"]), v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	} else if v.Type == "tenants" {
		var data map[string]any
		_ = json.Unmarshal(v.Data, &data)
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO tenants(id,workspace_id,identity,display_name,description,folder_id,data,code,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, folderID, v.Data, stringValue(data["code"]), v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	} else if v.Type == "projects" {
		data, navigationIdentity, relationErr := relationData(v.Data, "navigationIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		var navigationID *string
		if navigationIdentity != "" {
			resolved, resolveErr := r.resolveActiveDocumentID(ctx, v.WorkspaceID, "navigations", navigationIdentity)
			if resolveErr != nil {
				return nil, resolveErr
			}
			navigationID = &resolved
		}
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO projects(id,workspace_id,identity,display_name,description,folder_id,data,navigation_id,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, folderID, data, navigationID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	} else if v.Type == "updates" {
		data, storeIdentity, relationErr := relationData(v.Data, "storeIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		storeID, relationErr := r.resolveActiveDocumentID(ctx, v.WorkspaceID, "stores", storeIdentity)
		if relationErr != nil {
			return nil, relationErr
		}
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO updates(id,workspace_id,identity,display_name,description,folder_id,data,store_id,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, folderID, data, storeID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	} else if v.Type == "vocabs" {
		data, authProfileIdentity, relationErr := relationData(v.Data, "authProfileIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		var authProfileID *string
		if authProfileIdentity != "" {
			resolved, resolveErr := r.resolveActiveDocumentID(ctx, v.WorkspaceID, "auth-profiles", authProfileIdentity)
			if resolveErr != nil {
				return nil, resolveErr
			}
			authProfileID = &resolved
		}
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO vocabs(id,workspace_id,identity,display_name,description,folder_id,data,auth_profile_id,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, folderID, data, authProfileID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	} else {
		_, err = r.executor(ctx).Exec(ctx, `INSERT INTO `+table+`(id,workspace_id,identity,display_name,description,folder_id,data,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14)`, v.ID, v.WorkspaceID, v.Identity, v.DisplayName, v.Description, folderID, v.Data, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	}
	if err != nil {
		return nil, err
	}
	return r.GetDocument(ctx, v.WorkspaceID, v.Type, v.Identity, true)
}
func (r *EndgeRepository) UpdateDocument(ctx context.Context, v entities.Document, expected int, folderID *string) (*entities.Document, error) {
	table, err := tableFor(v.Type)
	if err != nil {
		return nil, err
	}
	var tag pgconn.CommandTag
	if v.Type == "folders" {
		var data map[string]any
		_ = json.Unmarshal(v.Data, &data)
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE folders SET identity=$1,display_name=$2,description=$3,entity_type=$4,parent_id=$5,is_root=$6,managed_by=$7,managed_by_id=$8,meta=$9,active=$10,deleted_at=$11,updated_by=$12,updated_at=NOW(),revision=revision+1 WHERE id=$13 AND workspace_id=$14 AND revision=$15`, v.Identity, v.DisplayName, v.Description, stringValue(data["entityType"]), folderID, boolValue(data["isRoot"]), v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	} else if v.Type == "tenants" {
		var data map[string]any
		_ = json.Unmarshal(v.Data, &data)
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE tenants SET identity=$1,display_name=$2,description=$3,folder_id=$4,data=$5,code=$6,managed_by=$7,managed_by_id=$8,meta=$9,active=$10,deleted_at=$11,updated_by=$12,updated_at=NOW(),revision=revision+1 WHERE id=$13 AND workspace_id=$14 AND revision=$15`, v.Identity, v.DisplayName, v.Description, folderID, v.Data, stringValue(data["code"]), v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	} else if v.Type == "projects" {
		data, navigationIdentity, relationErr := relationData(v.Data, "navigationIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		var navigationID *string
		if navigationIdentity != "" {
			resolved, resolveErr := r.resolveUpdatedDocumentRelation(ctx, v.WorkspaceID, "projects", "navigation_id", v.ID, "navigations", navigationIdentity)
			if resolveErr != nil {
				return nil, resolveErr
			}
			navigationID = &resolved
		}
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE projects SET identity=$1,display_name=$2,description=$3,folder_id=$4,data=$5,navigation_id=$6,managed_by=$7,managed_by_id=$8,meta=$9,active=$10,deleted_at=$11,updated_by=$12,updated_at=NOW(),revision=revision+1 WHERE id=$13 AND workspace_id=$14 AND revision=$15`, v.Identity, v.DisplayName, v.Description, folderID, data, navigationID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	} else if v.Type == "updates" {
		data, storeIdentity, relationErr := relationData(v.Data, "storeIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		storeID, relationErr := r.resolveUpdatedDocumentRelation(ctx, v.WorkspaceID, "updates", "store_id", v.ID, "stores", storeIdentity)
		if relationErr != nil {
			return nil, relationErr
		}
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE updates SET identity=$1,display_name=$2,description=$3,folder_id=$4,data=$5,store_id=$6,managed_by=$7,managed_by_id=$8,meta=$9,active=$10,deleted_at=$11,updated_by=$12,updated_at=NOW(),revision=revision+1 WHERE id=$13 AND workspace_id=$14 AND revision=$15`, v.Identity, v.DisplayName, v.Description, folderID, data, storeID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	} else if v.Type == "vocabs" {
		data, authProfileIdentity, relationErr := relationData(v.Data, "authProfileIdentity")
		if relationErr != nil {
			return nil, relationErr
		}
		var authProfileID *string
		if authProfileIdentity != "" {
			resolved, resolveErr := r.resolveUpdatedDocumentRelation(ctx, v.WorkspaceID, "vocabs", "auth_profile_id", v.ID, "auth-profiles", authProfileIdentity)
			if resolveErr != nil {
				return nil, resolveErr
			}
			authProfileID = &resolved
		}
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE vocabs SET identity=$1,display_name=$2,description=$3,folder_id=$4,data=$5,auth_profile_id=$6,managed_by=$7,managed_by_id=$8,meta=$9,active=$10,deleted_at=$11,updated_by=$12,updated_at=NOW(),revision=revision+1 WHERE id=$13 AND workspace_id=$14 AND revision=$15`, v.Identity, v.DisplayName, v.Description, folderID, data, authProfileID, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	} else {
		tag, err = r.executor(ctx).Exec(ctx, `UPDATE `+table+` SET identity=$1,display_name=$2,description=$3,folder_id=$4,data=$5,managed_by=$6,managed_by_id=$7,meta=$8,active=$9,deleted_at=$10,updated_by=$11,updated_at=NOW(),revision=revision+1 WHERE id=$12 AND workspace_id=$13 AND revision=$14`, v.Identity, v.DisplayName, v.Description, folderID, v.Data, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, v.WorkspaceID, expected)
	}
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetDocument(ctx, v.WorkspaceID, v.Type, v.Identity, true)
}

func (r *EndgeRepository) MoveFolderContents(ctx context.Context, workspaceID, folderID string, parentID *string, actor string) ([]entities.Document, error) {
	result := []entities.Document{}
	type movedIdentity struct{ kind, identity string }
	movedIdentities := []movedIdentity{}
	for kind, table := range documentTables {
		column := "folder_id"
		if kind == "folders" {
			column = "parent_id"
		}
		rows, err := r.executor(ctx).Query(ctx, `UPDATE `+table+` SET `+column+`=$1,updated_by=$2,updated_at=NOW(),revision=revision+1 WHERE workspace_id=$3 AND `+column+`=$4 RETURNING identity`, parentID, actor, workspaceID, folderID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var identity string
			if err := rows.Scan(&identity); err != nil {
				rows.Close()
				return nil, err
			}
			movedIdentities = append(movedIdentities, movedIdentity{kind: kind, identity: identity})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	for _, moved := range movedIdentities {
		value, err := r.GetDocument(ctx, workspaceID, moved.kind, moved.identity, true)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, nil
}
func (r *EndgeRepository) ResolveFolder(ctx context.Context, workspaceID, identity, entityType string) (*string, error) {
	if strings.TrimSpace(identity) == "" {
		return nil, nil
	}
	var id string
	err := r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM folders WHERE workspace_id=$1 AND identity=$2 AND entity_type=$3 AND deleted_at IS NULL`, workspaceID, identity, entityType).Scan(&id)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &id, nil
}

func (r *EndgeRepository) FolderWouldCycle(ctx context.Context, workspaceID, folderID, parentID string) (bool, error) {
	var cycle bool
	err := r.executor(ctx).QueryRow(ctx, `WITH RECURSIVE ancestors AS (
		SELECT id,parent_id FROM folders WHERE workspace_id=$1 AND id=$2
		UNION ALL
		SELECT f.id,f.parent_id FROM folders f JOIN ancestors a ON f.id=a.parent_id WHERE f.workspace_id=$1
	) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id=$3)`, workspaceID, parentID, folderID).Scan(&cycle)
	return cycle, err
}

func (r *EndgeRepository) ReplaceProjectEnvironments(ctx context.Context, document entities.Document, identities []string) error {
	if document.Type != "projects" {
		return fmt.Errorf("project environment relation requires a project document")
	}
	if _, err := r.executor(ctx).Exec(ctx, `DELETE FROM project_environments WHERE workspace_id=$1 AND project_id=$2`, document.WorkspaceID, document.ID); err != nil {
		return err
	}
	for index, identity := range identities {
		tag, err := r.executor(ctx).Exec(ctx, `INSERT INTO project_environments(workspace_id,project_id,environment_id,sort_order)
			SELECT $1,$2,target.id,$4 FROM environments target
			WHERE target.workspace_id=$1 AND target.identity=$3 AND target.deleted_at IS NULL`, document.WorkspaceID, document.ID, identity, index)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("relation target environments:%s not found", identity)
		}
	}
	return nil
}

func (r *EndgeRepository) resolveActiveDocumentID(ctx context.Context, workspaceID, kind, identity string) (string, error) {
	table, err := tableFor(kind)
	if err != nil {
		return "", err
	}
	var id string
	if err = r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM `+table+` WHERE workspace_id=$1 AND identity=$2 AND deleted_at IS NULL`, workspaceID, identity).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("relation target %s:%s not found", kind, identity)
		}
		return "", err
	}
	return id, nil
}

func (r *EndgeRepository) resolveUpdatedDocumentRelation(ctx context.Context, workspaceID, sourceTable, sourceColumn, sourceID, targetKind, identity string) (string, error) {
	targetTable, err := tableFor(targetKind)
	if err != nil {
		return "", err
	}
	var currentID, currentIdentity string
	err = r.executor(ctx).QueryRow(ctx, `SELECT target.id::text,target.identity FROM `+sourceTable+` source
		JOIN `+targetTable+` target ON target.workspace_id=source.workspace_id AND target.id=source.`+sourceColumn+`
		WHERE source.workspace_id=$1 AND source.id=$2`, workspaceID, sourceID).Scan(&currentID, &currentIdentity)
	if err == nil && currentIdentity == identity {
		return currentID, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return r.resolveActiveDocumentID(ctx, workspaceID, targetKind, identity)
}

func relationData(raw json.RawMessage, field string) (json.RawMessage, string, error) {
	data := map[string]any{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, "", fmt.Errorf("document data is invalid: %w", err)
	}
	identity := strings.TrimSpace(stringValue(data[field]))
	delete(data, field)
	return mustJSON(data), identity, nil
}
