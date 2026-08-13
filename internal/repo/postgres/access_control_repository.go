package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

const accessBootstrapLockID int64 = 0x454e44474541434c

func (r *EndgeRepository) LockBootstrap(ctx context.Context) error {
	_, err := r.executor(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, accessBootstrapLockID)
	return err
}

func (r *EndgeRepository) HasPlatformAdmins(ctx context.Context) (bool, error) {
	var value bool
	err := r.executor(ctx).QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM access_grants g JOIN service_users u ON u.id=g.user_id
		WHERE g.scope_type='platform' AND u.active=TRUE AND u.is_system=FALSE
	)`).Scan(&value)
	return value, err
}

func (r *EndgeRepository) IsPlatformAdmin(ctx context.Context, userID string) (bool, error) {
	var value bool
	err := r.executor(ctx).QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM access_grants WHERE scope_type='platform' AND user_id=$1
	)`, userID).Scan(&value)
	return value, err
}

func (r *EndgeRepository) CountHumanUsers(ctx context.Context) (int, error) {
	var value int
	err := r.executor(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM service_users WHERE active=TRUE AND is_system=FALSE`).Scan(&value)
	return value, err
}

func (r *EndgeRepository) SearchServiceUsers(ctx context.Context, input ports.ServiceUserSearchInput) (entities.ServiceUserPage, error) {
	args := []any{strings.ToLower(strings.TrimSpace(input.Query)) + "%"}
	where := `u.active=TRUE AND u.is_system=FALSE AND lower(u.username) LIKE $1`
	if input.Cursor != nil {
		args = append(args, strings.ToLower(input.Cursor.Username), input.Cursor.ID)
		where += fmt.Sprintf(` AND (lower(u.username),u.id) > ($%d,$%d::uuid)`, len(args)-1, len(args))
	}
	args = append(args, input.Limit+1)
	rows, err := r.executor(ctx).Query(ctx, fmt.Sprintf(`SELECT u.id::text,u.provider_id,u.username,u.display_name,u.active
		FROM service_users u WHERE %s ORDER BY lower(u.username),u.id LIMIT $%d`, where, len(args)), args...)
	if err != nil {
		return entities.ServiceUserPage{}, err
	}
	defer rows.Close()
	items := make([]entities.AccessGrantUser, 0, input.Limit+1)
	for rows.Next() {
		var item entities.AccessGrantUser
		if err := rows.Scan(&item.ID, &item.ProviderID, &item.Username, &item.DisplayName, &item.Active); err != nil {
			return entities.ServiceUserPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return entities.ServiceUserPage{}, err
	}
	page := entities.ServiceUserPage{Items: items}
	if len(items) > input.Limit {
		last := items[input.Limit-1]
		page.Items = items[:input.Limit]
		page.Next = &entities.AccessGrantCursor{Username: last.Username, ID: last.ID}
	}
	return page, nil
}

func (r *EndgeRepository) GetAccessGrant(ctx context.Context, id string) (*entities.AccessGrant, error) {
	return scanAccessGrant(r.executor(ctx).QueryRow(ctx, accessGrantSelect+` WHERE g.id=$1`, id))
}

func (r *EndgeRepository) ListAccessGrants(ctx context.Context, input ports.AccessGrantListInput) (entities.AccessGrantPage, error) {
	args := []any{input.ScopeType}
	where := `g.scope_type=$1`
	if input.WorkspaceID != nil {
		args = append(args, *input.WorkspaceID)
		where += fmt.Sprintf(` AND g.workspace_id=$%d`, len(args))
	}
	if input.UserID != nil {
		args = append(args, *input.UserID)
		where += fmt.Sprintf(` AND g.user_id=$%d`, len(args))
	}
	if query := strings.ToLower(strings.TrimSpace(input.Query)); query != "" {
		args = append(args, query+"%")
		where += fmt.Sprintf(` AND lower(u.username) LIKE $%d`, len(args))
	}
	if input.Cursor != nil {
		args = append(args, strings.ToLower(input.Cursor.Username), input.Cursor.ID)
		where += fmt.Sprintf(` AND (lower(u.username),g.id) > ($%d,$%d::uuid)`, len(args)-1, len(args))
	}
	args = append(args, input.Limit+1)
	rows, err := r.executor(ctx).Query(ctx, fmt.Sprintf(`%s WHERE %s ORDER BY lower(u.username),g.id LIMIT $%d`, accessGrantSelect, where, len(args)), args...)
	if err != nil {
		return entities.AccessGrantPage{}, err
	}
	defer rows.Close()
	items := make([]entities.AccessGrant, 0, input.Limit+1)
	for rows.Next() {
		item, scanErr := scanAccessGrant(rows)
		if scanErr != nil {
			return entities.AccessGrantPage{}, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return entities.AccessGrantPage{}, err
	}
	page := entities.AccessGrantPage{Items: items}
	if len(items) > input.Limit {
		last := items[input.Limit-1]
		page.Items = items[:input.Limit]
		page.Next = &entities.AccessGrantCursor{Username: last.User.Username, ID: last.ID}
	}
	return page, nil
}

func (r *EndgeRepository) UpsertAccessGrant(ctx context.Context, input ports.AccessGrantInput) (*entities.AccessGrant, bool, error) {
	var existingID string
	if input.ScopeType == "platform" {
		_ = r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM access_grants WHERE scope_type='platform' AND user_id=$1`, input.UserID).Scan(&existingID)
		created := existingID == ""
		row := r.executor(ctx).QueryRow(ctx, `INSERT INTO access_grants(user_id,scope_type,role,created_by,updated_by)
			SELECT id,'platform','admin',$2,$2 FROM service_users WHERE id=$1 AND active=TRUE AND is_system=FALSE
			ON CONFLICT(user_id) WHERE scope_type='platform' DO UPDATE SET role='admin',updated_by=EXCLUDED.updated_by,updated_at=NOW()
			RETURNING id::text`, input.UserID, input.ActorID)
		if err := row.Scan(&existingID); err != nil {
			return nil, false, repositoryError(err)
		}
		grant, err := r.GetAccessGrant(ctx, existingID)
		return grant, created, err
	} else {
		if input.WorkspaceID == nil {
			return nil, false, fmt.Errorf("workspace id is required")
		}
		_ = r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM access_grants WHERE scope_type='workspace' AND workspace_id=$1 AND user_id=$2`, *input.WorkspaceID, input.UserID).Scan(&existingID)
		created := existingID == ""
		row := r.executor(ctx).QueryRow(ctx, `INSERT INTO access_grants(user_id,scope_type,workspace_id,role,created_by,updated_by)
			SELECT u.id,'workspace',$2,$3,$4,$4 FROM service_users u JOIN workspaces w ON w.id=$2
			WHERE u.id=$1 AND u.active=TRUE AND u.is_system=FALSE
			ON CONFLICT(workspace_id,user_id) WHERE scope_type='workspace' DO UPDATE SET role=EXCLUDED.role,updated_by=EXCLUDED.updated_by,updated_at=NOW()
			RETURNING id::text`, input.UserID, *input.WorkspaceID, input.Role, input.ActorID)
		if err := row.Scan(&existingID); err != nil {
			return nil, false, repositoryError(err)
		}
		_, err := r.executor(ctx).Exec(ctx, `INSERT INTO workspace_memberships(workspace_id,user_id,role,created_by)
			VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role,updated_at=NOW()`, *input.WorkspaceID, input.UserID, input.Role, input.ActorID)
		if err != nil {
			return nil, false, err
		}
		grant, err := r.GetAccessGrant(ctx, existingID)
		return grant, created, err
	}
	return nil, false, fmt.Errorf("unsupported access scope %q", input.ScopeType)
}

func (r *EndgeRepository) DeleteAccessGrant(ctx context.Context, id string) error {
	grant, err := r.GetAccessGrant(ctx, id)
	if err != nil {
		return err
	}
	if grant.ScopeType == "workspace" && grant.WorkspaceID != nil {
		if _, err = r.executor(ctx).Exec(ctx, `DELETE FROM workspace_memberships WHERE workspace_id=$1 AND user_id=$2`, *grant.WorkspaceID, grant.User.ID); err != nil {
			return err
		}
	}
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM access_grants WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *EndgeRepository) CountPlatformAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.executor(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM access_grants g JOIN service_users u ON u.id=g.user_id
		WHERE g.scope_type='platform' AND u.active=TRUE AND u.is_system=FALSE`).Scan(&count)
	return count, err
}

const accessGrantSelect = `SELECT g.id::text,g.scope_type,g.workspace_id::text,w.identity,w.display_name,g.role,
	u.id::text,u.provider_id,u.username,u.display_name,u.active,
	` + `jsonb_build_object('id',cu.id::text,'username',cu.username,'displayName',cu.display_name),` +
	`jsonb_build_object('id',uu.id::text,'username',uu.username,'displayName',uu.display_name),g.created_at,g.updated_at
	FROM access_grants g JOIN service_users u ON u.id=g.user_id
	LEFT JOIN workspaces w ON w.id=g.workspace_id
	JOIN service_users cu ON cu.id=g.created_by JOIN service_users uu ON uu.id=g.updated_by`

func scanAccessGrant(row scanner) (*entities.AccessGrant, error) {
	value := &entities.AccessGrant{}
	var created, updated []byte
	if err := row.Scan(&value.ID, &value.ScopeType, &value.WorkspaceID, &value.WorkspaceIdentity, &value.WorkspaceDisplayName,
		&value.Role, &value.User.ID, &value.User.ProviderID, &value.User.Username, &value.User.DisplayName, &value.User.Active,
		&created, &updated, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(created, &value.CreatedBy)
	_ = json.Unmarshal(updated, &value.UpdatedBy)
	return value, nil
}
