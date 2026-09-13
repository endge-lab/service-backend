package postgres

import (
	"context"
	"fmt"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func (r *EndgeRepository) ListBuildProfilesForTransfer(ctx context.Context, workspaceID, actorID, privateMode string) ([]entities.BuildProfile, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT `+buildProfileColumns+buildProfileJoins+`
        WHERE p.workspace_id=$1 AND (
            p.visibility='shared' OR $3='all' OR ($3='own' AND p.owner_user_id=$2)
        ) ORDER BY p.visibility,lower(p.display_name),p.identity`, workspaceID, actorID, privateMode)
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

func (r *EndgeRepository) ListAIConnectionsForTransfer(ctx context.Context, actorID, privateMode string, includePublic bool) ([]ports.AITransferConnection, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT c.id::text,c.name,c.adapter,c.base_url,c.visibility,
        COALESCE(c.owner_user_id::text,''),COALESCE(owner.username,''),c.credential_encrypted,c.enabled,
        c.created_by::text,c.updated_by::text,c.created_at,c.updated_at
        FROM ai_provider_connections c LEFT JOIN service_users owner ON owner.id=c.owner_user_id
        WHERE ($3 AND c.visibility='public') OR (
            c.visibility='private' AND ($2='all' OR ($2='own' AND c.owner_user_id=$1))
        ) ORDER BY c.visibility,c.adapter,lower(c.name),c.id`, actorID, privateMode, includePublic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ports.AITransferConnection, 0)
	for rows.Next() {
		var item ports.AITransferConnection
		if err = rows.Scan(&item.Connection.ID, &item.Connection.Name, &item.Connection.Adapter, &item.Connection.BaseURL,
			&item.Connection.Visibility, &item.Connection.OwnerUserID, &item.OwnerLogin, &item.Credential,
			&item.Connection.Enabled, &item.Connection.CreatedBy, &item.Connection.UpdatedBy,
			&item.Connection.CreatedAt, &item.Connection.UpdatedAt); err != nil {
			return nil, repositoryError(err)
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for index := range result {
		models, modelErr := r.listAIModelsForTransfer(ctx, result[index].Connection.ID)
		if modelErr != nil {
			return nil, modelErr
		}
		result[index].Models = models
	}
	return result, nil
}

func (r *EndgeRepository) listAIModelsForTransfer(ctx context.Context, connectionID string) ([]entities.AIModelProfile, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT id::text,connection_id::text,provider_model_id,display_name,enabled,is_default,
        created_by::text,updated_by::text,created_at,updated_at
        FROM ai_model_profiles WHERE connection_id=$1 ORDER BY lower(display_name),id`, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]entities.AIModelProfile, 0)
	for rows.Next() {
		var value entities.AIModelProfile
		if err = rows.Scan(&value.ID, &value.ConnectionID, &value.ProviderModelID, &value.DisplayName,
			&value.Enabled, &value.Default, &value.CreatedBy, &value.UpdatedBy, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, repositoryError(err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) FindUserIDsByNormalizedLogin(ctx context.Context, login string) ([]string, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT id::text FROM service_users
        WHERE lower(btrim(username))=lower(btrim($1)) ORDER BY id`, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) UpsertBuildProfileForTransfer(ctx context.Context, value entities.BuildProfile) error {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO build_profiles(
        id,identity,workspace_id,display_name,visibility,owner_user_id,settings_version,settings,created_by,updated_by
    ) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
    ON CONFLICT(workspace_id,identity) DO UPDATE SET
        display_name=excluded.display_name,visibility=excluded.visibility,owner_user_id=excluded.owner_user_id,
        settings_version=excluded.settings_version,settings=excluded.settings,revision=build_profiles.revision+1,
        updated_by=excluded.updated_by,updated_at=now()`,
		value.ID, value.Identity, value.WorkspaceID, value.DisplayName, value.Visibility, value.OwnerUserID,
		value.SettingsVersion, value.Settings, value.UpdatedBy.ID)
	return err
}

func (r *EndgeRepository) FindAIConnectionIDForTransfer(ctx context.Context, visibility, ownerUserID, adapter, name string) (string, error) {
	query := `SELECT id::text FROM ai_provider_connections WHERE visibility='public' AND adapter=$1 AND name=$2`
	args := []any{adapter, name}
	if visibility == entities.AIVisibilityPrivate {
		query = `SELECT id::text FROM ai_provider_connections WHERE visibility='private' AND owner_user_id=$1 AND adapter=$2 AND name=$3`
		args = []any{ownerUserID, adapter, name}
	}
	var id string
	err := r.executor(ctx).QueryRow(ctx, query, args...).Scan(&id)
	return id, repositoryError(err)
}

func (r *EndgeRepository) UpsertAIConnectionForTransfer(ctx context.Context, value entities.AIProviderConnection, credential []byte) (string, error) {
	conflictTarget := `(adapter,name) WHERE visibility='public'`
	if value.Visibility == entities.AIVisibilityPrivate {
		conflictTarget = `(owner_user_id,adapter,name) WHERE visibility='private'`
	}
	query := fmt.Sprintf(`INSERT INTO ai_provider_connections(
        id,name,adapter,base_url,visibility,owner_user_id,credential_encrypted,enabled,created_by,updated_by
    ) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
    ON CONFLICT %s DO UPDATE SET base_url=excluded.base_url,credential_encrypted=excluded.credential_encrypted,
        enabled=excluded.enabled,updated_by=excluded.updated_by,updated_at=now() RETURNING id::text`, conflictTarget)
	var id string
	err := r.executor(ctx).QueryRow(ctx, query, value.ID, value.Name, value.Adapter, value.BaseURL,
		value.Visibility, nullableAIString(value.OwnerUserID), nullableAIBytes(credential), value.Enabled, value.UpdatedBy).Scan(&id)
	return id, err
}

func (r *EndgeRepository) UpsertAIModelForTransfer(ctx context.Context, value entities.AIModelProfile) error {
	if value.Default {
		if _, err := r.executor(ctx).Exec(ctx, `UPDATE ai_model_profiles SET is_default=false,updated_by=$1,updated_at=now()
            WHERE is_default=true AND id<>$2`, value.UpdatedBy, value.ID); err != nil {
			return err
		}
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO ai_model_profiles(
        id,connection_id,provider_model_id,display_name,enabled,is_default,created_by,updated_by
    ) VALUES($1,$2,$3,$4,$5,$6,$7,$7)
    ON CONFLICT(connection_id,provider_model_id) DO UPDATE SET display_name=excluded.display_name,
        enabled=excluded.enabled,is_default=excluded.is_default,updated_by=excluded.updated_by,updated_at=now()`,
		value.ID, value.ConnectionID, value.ProviderModelID, value.DisplayName, value.Enabled, value.Default, value.UpdatedBy)
	return err
}
