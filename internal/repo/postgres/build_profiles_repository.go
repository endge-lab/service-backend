package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

const buildProfileColumns = `p.id::text,p.identity::text,p.workspace_id::text,p.display_name,p.visibility,p.owner_user_id::text,owner.username,
    p.settings_version,p.settings,p.revision,` +
	`jsonb_build_object('id',created.id::text,'username',created.username,'displayName',created.display_name),` +
	`jsonb_build_object('id',updated.id::text,'username',updated.username,'displayName',updated.display_name),p.created_at,p.updated_at`

const buildProfileActorJoins = ` JOIN service_users owner ON owner.id=p.owner_user_id
    JOIN service_users created ON created.id=p.created_by
    JOIN service_users updated ON updated.id=p.updated_by`

const buildProfileJoins = ` FROM build_profiles p` + buildProfileActorJoins

func scanBuildProfile(row scanner) (*entities.BuildProfile, error) {
	value := &entities.BuildProfile{}
	var created, updated []byte
	if err := row.Scan(&value.ID, &value.Identity, &value.WorkspaceID, &value.DisplayName, &value.Visibility,
		&value.OwnerUserID, &value.OwnerLogin, &value.SettingsVersion, &value.Settings, &value.Revision,
		&created, &updated, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = jsonUnmarshalActor(created, &value.CreatedBy)
	_ = jsonUnmarshalActor(updated, &value.UpdatedBy)
	return value, nil
}

func jsonUnmarshalActor(raw []byte, target *entities.Actor) error {
	return json.Unmarshal(raw, target)
}

func (r *EndgeRepository) ListBuildProfiles(ctx context.Context, workspaceID, actorID string) ([]entities.BuildProfile, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT `+buildProfileColumns+buildProfileJoins+`
        WHERE p.workspace_id=$1 AND (p.visibility='shared' OR p.owner_user_id=$2)
        ORDER BY CASE p.visibility WHEN 'shared' THEN 0 ELSE 1 END,lower(p.display_name),p.identity`, workspaceID, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]entities.BuildProfile, 0)
	for rows.Next() {
		value, scanErr := scanBuildProfile(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetBuildProfile(ctx context.Context, workspaceID, identity string) (*entities.BuildProfile, error) {
	return scanBuildProfile(r.executor(ctx).QueryRow(ctx, `SELECT `+buildProfileColumns+buildProfileJoins+`
        WHERE p.workspace_id=$1 AND p.identity=$2`, workspaceID, identity))
}

func (r *EndgeRepository) LockBuildProfileNames(ctx context.Context, workspaceID string) error {
	var id string
	return r.executor(ctx).QueryRow(ctx, `SELECT id::text FROM workspaces WHERE id=$1 FOR UPDATE`, workspaceID).Scan(&id)
}

func (r *EndgeRepository) NextBuildProfileNumber(ctx context.Context, workspaceID string) (int, error) {
	var value int
	err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(max((regexp_match(display_name, '^Новый профиль ([0-9]+)$'))[1]::int),0)+1
        FROM build_profiles WHERE workspace_id=$1`, workspaceID).Scan(&value)
	return value, err
}

func (r *EndgeRepository) InsertBuildProfile(ctx context.Context, value entities.BuildProfile) (*entities.BuildProfile, error) {
	return scanBuildProfile(r.executor(ctx).QueryRow(ctx, `WITH inserted AS (
        INSERT INTO build_profiles(id,identity,workspace_id,display_name,visibility,owner_user_id,settings_version,settings,created_by,updated_by)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) RETURNING *
	    ) SELECT `+buildProfileColumns+` FROM inserted p`+buildProfileActorJoins,
		value.ID, value.Identity, value.WorkspaceID, value.DisplayName, value.Visibility, value.OwnerUserID,
		value.SettingsVersion, value.Settings, value.CreatedBy.ID))
}

func (r *EndgeRepository) UpdateBuildProfile(ctx context.Context, value entities.BuildProfile, expected int) (*entities.BuildProfile, error) {
	updated, err := scanBuildProfile(r.executor(ctx).QueryRow(ctx, `WITH changed AS (
        UPDATE build_profiles SET display_name=$3,visibility=$4,settings_version=$5,settings=$6,
            revision=revision+1,updated_by=$7,updated_at=now()
        WHERE workspace_id=$1 AND identity=$2 AND revision=$8 RETURNING *
    ) SELECT `+buildProfileColumns+` FROM changed p`+buildProfileActorJoins,
		value.WorkspaceID, value.Identity, value.DisplayName, value.Visibility, value.SettingsVersion,
		value.Settings, value.UpdatedBy.ID, expected))
	if errors.Is(err, ports.ErrNotFound) {
		return nil, errors.New("revision conflict")
	}
	return updated, err
}

func (r *EndgeRepository) DeleteBuildProfile(ctx context.Context, workspaceID, identity string, expected int) error {
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM build_profiles WHERE workspace_id=$1 AND identity=$2 AND revision=$3`, workspaceID, identity, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("revision conflict")
	}
	return nil
}
