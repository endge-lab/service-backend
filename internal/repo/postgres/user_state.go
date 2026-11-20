package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/jackc/pgx/v5"
)

func (r *EndgeRepository) LockUserState(ctx context.Context, s entities.UserStateScope, user string) error {
	owner := s.WorkspaceID
	if s.Application != "" {
		owner = "app:" + s.Application + ":" + user
	}
	_, err := r.executor(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('user-state:' || $1::text || ':' || $2::text || ':' || $3::text, 0))`, owner, s.Environment, s.Facets)
	return err
}

func (r *EndgeRepository) UserStateScopes(ctx context.Context, s entities.UserStateScope) ([]string, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT DISTINCT facet_scope FROM user_state
		WHERE workspace_id=$1 AND environment=$2 AND value IS NOT NULL ORDER BY facet_scope`, s.WorkspaceID, s.Environment)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *EndgeRepository) UserStateUsers(ctx context.Context, s entities.UserStateScope, search, after string, limit int) ([]entities.UserStateUser, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT u.id::text,u.username,u.display_name,count(*)
		FROM user_state p JOIN service_users u ON u.id=p.user_id
		WHERE p.workspace_id=$1 AND p.environment=$2 AND p.facet_scope=$3 AND p.value IS NOT NULL AND u.active
		AND u.id > COALESCE(NULLIF($5,'')::uuid,'00000000-0000-0000-0000-000000000000'::uuid)
		AND ($4='' OR strpos(lower(u.username),lower($4))>0 OR strpos(lower(u.display_name),lower($4))>0)
		GROUP BY u.id ORDER BY u.id LIMIT $6`, s.WorkspaceID, s.Environment, s.Facets, search, after, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[entities.UserStateUser])
}

func (r *EndgeRepository) UserStateRevision(ctx context.Context, s entities.UserStateScope, user string) (int64, error) {
	if s.Application != "" {
		return 0, nil
	}
	var revision int64
	err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(MAX(revision),0) FROM user_state_resets
		WHERE workspace_id=$1 AND environment=$2 AND facet_scope=$3 AND user_id IN ('',$4)`, s.WorkspaceID, s.Environment, s.Facets, user).Scan(&revision)
	return revision, err
}

const userStateColumns = `key,value,value IS NULL,revision,updated_at,COALESCE(updated_by::text,''),mutation_id`

func scanUserState(row scanner) (entities.UserStateEntry, error) {
	var item entities.UserStateEntry
	err := row.Scan(&item.Key, &item.Value, &item.Deleted, &item.Revision, &item.UpdatedAt, &item.UpdatedBy, &item.MutationID)
	return item, err
}

func (r *EndgeRepository) ReadUserState(ctx context.Context, s entities.UserStateScope, user string) ([]entities.UserStateEntry, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT `+userStateColumns+` FROM user_state
		WHERE (workspace_id=NULLIF($1,'')::uuid OR ($1='' AND workspace_id IS NULL))
		AND environment=$2 AND facet_scope=$3 AND user_id=$4 AND application=$5 ORDER BY key`, s.WorkspaceID, s.Environment, s.Facets, user, s.Application)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entities.UserStateEntry, error) { return scanUserState(row) })
}

func (r *EndgeRepository) WriteUserState(ctx context.Context, s entities.UserStateScope, user, actor string, item entities.UserStateEntry) (entities.UserStateEntry, error) {
	var value any
	if !item.Deleted {
		value = item.Value
	}
	return scanUserState(r.executor(ctx).QueryRow(ctx, `INSERT INTO user_state
		(workspace_id,environment,facet_scope,user_id,key,value,mutation_id,updated_by,application)
		VALUES(NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT ON CONSTRAINT user_state_key DO UPDATE SET
			value=excluded.value, revision=excluded.revision, mutation_id=excluded.mutation_id,
			updated_at=now(), updated_by=excluded.updated_by
		RETURNING `+userStateColumns, s.WorkspaceID, s.Environment, s.Facets, user, item.Key, value, item.MutationID, actor, s.Application))
}

func (r *EndgeRepository) ResetUserState(ctx context.Context, s entities.UserStateScope, user string) (int64, error) {
	db := r.executor(ctx)
	var count int64
	err := db.QueryRow(ctx, `WITH removed AS (
		DELETE FROM user_state WHERE workspace_id=$1 AND environment=$2 AND facet_scope=$3
		AND ($4='' OR user_id=NULLIF($4,'')::uuid) RETURNING value
	) SELECT count(*) FROM removed WHERE value IS NOT NULL`, s.WorkspaceID, s.Environment, s.Facets, user).Scan(&count)
	if err != nil {
		return 0, err
	}
	_, err = db.Exec(ctx, `INSERT INTO user_state_resets(workspace_id,environment,facet_scope,user_id) VALUES($1,$2,$3,$4)
		ON CONFLICT(workspace_id,environment,facet_scope,user_id) DO UPDATE SET revision=excluded.revision`, s.WorkspaceID, s.Environment, s.Facets, user)
	return count, err
}

func (r *EndgeRepository) RecordUserStateAdminAction(ctx context.Context, s entities.UserStateScope, user, actor, action string, keys []string, count int64) error {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO user_state_admin_events
		(workspace_id,environment,facet_scope,target_user_id,actor_id,action,affected_keys,affected_count)
		VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8)`, s.WorkspaceID, s.Environment, s.Facets, user, actor, action, keys, count)
	return err
}
