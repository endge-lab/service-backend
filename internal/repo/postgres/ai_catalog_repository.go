package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

const aiProviderConnectionColumns = `c.id::text,c.name,c.adapter,c.base_url,c.credential_encrypted IS NOT NULL,c.enabled,
    (SELECT count(*) FROM ai_model_profiles p WHERE p.connection_id=c.id)::int,
    c.created_by::text,c.updated_by::text,c.created_at,c.updated_at`

const aiModelProfileColumns = `p.id::text,p.connection_id::text,c.name,c.adapter,p.provider_model_id,p.display_name,
    p.enabled,p.is_default,p.created_by::text,p.updated_by::text,p.created_at,p.updated_at`

func scanAIProviderConnection(row scanner) (*entities.AIProviderConnection, error) {
	value := &entities.AIProviderConnection{}
	if err := row.Scan(&value.ID, &value.Name, &value.Adapter, &value.BaseURL, &value.HasCredential, &value.Enabled,
		&value.ModelCount, &value.CreatedBy, &value.UpdatedBy, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	return value, nil
}

func scanAIModelProfile(row scanner) (*entities.AIModelProfile, error) {
	value := &entities.AIModelProfile{}
	if err := row.Scan(&value.ID, &value.ConnectionID, &value.ConnectionName, &value.Adapter,
		&value.ProviderModelID, &value.DisplayName, &value.Enabled, &value.Default,
		&value.CreatedBy, &value.UpdatedBy, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	return value, nil
}

func (r *EndgeRepository) ListAIProviderConnections(ctx context.Context) ([]entities.AIProviderConnection, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT `+aiProviderConnectionColumns+` FROM ai_provider_connections c ORDER BY lower(c.name),c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]entities.AIProviderConnection, 0)
	for rows.Next() {
		value, err := scanAIProviderConnection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetAIProviderConnection(ctx context.Context, id string) (*ports.AIProviderConnectionRecord, error) {
	row := r.executor(ctx).QueryRow(ctx, `SELECT `+aiProviderConnectionColumns+`,c.credential_encrypted FROM ai_provider_connections c WHERE c.id=$1`, id)
	record := &ports.AIProviderConnectionRecord{}
	if err := row.Scan(&record.Connection.ID, &record.Connection.Name, &record.Connection.Adapter, &record.Connection.BaseURL,
		&record.Connection.HasCredential, &record.Connection.Enabled, &record.Connection.ModelCount,
		&record.Connection.CreatedBy, &record.Connection.UpdatedBy, &record.Connection.CreatedAt, &record.Connection.UpdatedAt,
		&record.Credential); err != nil {
		return nil, repositoryError(err)
	}
	return record, nil
}

func (r *EndgeRepository) InsertAIProviderConnection(ctx context.Context, value entities.AIProviderConnection, credential []byte) (*entities.AIProviderConnection, error) {
	return scanAIProviderConnection(r.executor(ctx).QueryRow(ctx, `INSERT INTO ai_provider_connections AS c(
		id,name,adapter,base_url,credential_encrypted,enabled,created_by,updated_by
	) VALUES($1,$2,$3,$4,$5,$6,$7,$7) RETURNING `+aiProviderConnectionColumns,
		value.ID, value.Name, value.Adapter, value.BaseURL, nullableAIBytes(credential), value.Enabled, value.CreatedBy))
}

func (r *EndgeRepository) UpdateAIProviderConnection(ctx context.Context, value entities.AIProviderConnection) (*entities.AIProviderConnection, error) {
	return scanAIProviderConnection(r.executor(ctx).QueryRow(ctx, `UPDATE ai_provider_connections c
		SET name=$2,base_url=$3,enabled=$4,updated_by=$5,updated_at=now() WHERE c.id=$1 RETURNING `+aiProviderConnectionColumns,
		value.ID, value.Name, value.BaseURL, value.Enabled, value.UpdatedBy))
}

func (r *EndgeRepository) UpdateAIProviderCredential(ctx context.Context, id, actorID string, credential []byte) (*entities.AIProviderConnection, error) {
	return scanAIProviderConnection(r.executor(ctx).QueryRow(ctx, `UPDATE ai_provider_connections c
		SET credential_encrypted=$2,updated_by=$3,updated_at=now() WHERE c.id=$1 RETURNING `+aiProviderConnectionColumns,
		id, credential, actorID))
}

func (r *EndgeRepository) DeleteAIProviderConnection(ctx context.Context, id string) error {
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM ai_provider_connections WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *EndgeRepository) ListAIModelProfiles(ctx context.Context, enabledOnly bool) ([]entities.AIModelProfile, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT `+aiModelProfileColumns+` FROM ai_model_profiles p
		JOIN ai_provider_connections c ON c.id=p.connection_id
		WHERE NOT $1 OR (p.enabled AND c.enabled)
		ORDER BY p.is_default DESC,lower(p.display_name),p.id`, enabledOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]entities.AIModelProfile, 0)
	for rows.Next() {
		value, err := scanAIModelProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetAIModelProfile(ctx context.Context, id string) (*entities.AIModelProfile, error) {
	return scanAIModelProfile(r.executor(ctx).QueryRow(ctx, `SELECT `+aiModelProfileColumns+` FROM ai_model_profiles p
		JOIN ai_provider_connections c ON c.id=p.connection_id WHERE p.id=$1`, id))
}

func (r *EndgeRepository) InsertAIModelProfile(ctx context.Context, value entities.AIModelProfile) (*entities.AIModelProfile, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO ai_model_profiles(
		id,connection_id,provider_model_id,display_name,enabled,is_default,created_by,updated_by
	) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`,
		value.ID, value.ConnectionID, value.ProviderModelID, value.DisplayName, value.Enabled, value.Default, value.CreatedBy)
	if err != nil {
		return nil, err
	}
	return r.GetAIModelProfile(ctx, value.ID)
}

func (r *EndgeRepository) UpdateAIModelProfile(ctx context.Context, value entities.AIModelProfile) (*entities.AIModelProfile, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE ai_model_profiles p SET
		provider_model_id=$2,display_name=$3,enabled=$4,is_default=$5,updated_by=$6,updated_at=now()
		WHERE p.id=$1`, value.ID, value.ProviderModelID, value.DisplayName, value.Enabled, value.Default, value.UpdatedBy)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}
	return r.GetAIModelProfile(ctx, value.ID)
}

func nullableAIBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func (r *EndgeRepository) ClearAIModelDefaults(ctx context.Context, exceptID string) error {
	_, err := r.executor(ctx).Exec(ctx, `UPDATE ai_model_profiles SET is_default=false,updated_at=now() WHERE is_default AND id<>$1`, exceptID)
	return err
}

func (r *EndgeRepository) DeleteAIModelProfile(ctx context.Context, id string) error {
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM ai_model_profiles WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}
