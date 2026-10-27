package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

func accessTokenSelect() string {
	return accessTokenSelectWith("")
}

// accessTokenSelectWith добавляет extraColumns в конец списка, сохраняя порядок scanAccessToken.
func accessTokenSelectWith(extraColumns string) string {
	return `SELECT t.id::text,` + actorScan("owner") + `,t.workspace_id::text,w.identity,w.display_name,t.name,t.description,t.scope,
		t.can_commit,t.can_delete,t.token_prefix,t.created_at,t.expires_at,t.last_used_at,t.revoked_at,
		CASE WHEN revoker.id IS NULL THEN NULL ELSE ` + actorScan("revoker") + ` END` + extraColumns + `
		FROM access_tokens t
		JOIN service_users owner ON owner.id=t.user_id
		JOIN workspaces w ON w.id=t.workspace_id
		LEFT JOIN service_users revoker ON revoker.id=t.revoked_by`
}

func scanAccessToken(row scanner, extra ...any) (*entities.AccessToken, error) {
	value := &entities.AccessToken{}
	var owner []byte
	var revoker []byte
	targets := []any{&value.ID, &owner, &value.WorkspaceID, &value.WorkspaceIdentity, &value.WorkspaceDisplayName, &value.Name,
		&value.Description, &value.Scope, &value.CanCommit, &value.CanDelete, &value.TokenPrefix, &value.CreatedAt,
		&value.ExpiresAt, &value.LastUsedAt, &value.RevokedAt, &revoker}
	if err := row.Scan(append(targets, extra...)...); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(owner, &value.Owner)
	if len(revoker) > 0 {
		value.RevokedBy = &entities.Actor{}
		_ = json.Unmarshal(revoker, value.RevokedBy)
	}
	return value, nil
}

func (r *EndgeRepository) CreateAccessToken(ctx context.Context, input ports.CreateAccessTokenInput) (*entities.AccessToken, error) {
	id := uuid.NewString()
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO access_tokens(id,user_id,workspace_id,name,description,scope,can_commit,can_delete,token_prefix,secret_hash,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, input.UserID, input.WorkspaceID, input.Name, input.Description, input.Scope,
		input.CanCommit, input.CanDelete, input.TokenPrefix, input.SecretHash, input.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return r.GetAccessToken(ctx, id)
}

func (r *EndgeRepository) GetAccessToken(ctx context.Context, id string) (*entities.AccessToken, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ports.ErrNotFound
	}
	return scanAccessToken(r.executor(ctx).QueryRow(ctx, accessTokenSelect()+` WHERE t.id=$1`, id))
}

func (r *EndgeRepository) ListAccessTokens(ctx context.Context, userID *string) ([]entities.AccessToken, error) {
	rows, err := r.executor(ctx).Query(ctx, accessTokenSelect()+` WHERE ($1::uuid IS NULL OR t.user_id=$1::uuid) ORDER BY t.created_at DESC,t.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []entities.AccessToken{}
	for rows.Next() {
		value, scanErr := scanAccessToken(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *value)
	}
	return items, rows.Err()
}

// FindAccessTokenCredential ищет токен по уникальному хэшу и сразу возвращает владельца,
// чтобы middleware не делал отдельный запрос пользователя.
func (r *EndgeRepository) FindAccessTokenCredential(ctx context.Context, secretHash []byte) (*ports.AccessTokenCredential, error) {
	credential := &ports.AccessTokenCredential{}
	owner := &credential.Owner
	query := accessTokenSelectWith(`,t.secret_hash,owner.id::text,owner.provider_id,owner.subject,owner.issuer,owner.username,
		owner.display_name,owner.active,owner.last_seen_at,owner.created_at,owner.updated_at`) + ` WHERE t.secret_hash=$1`
	token, err := scanAccessToken(r.executor(ctx).QueryRow(ctx, query, secretHash), &credential.SecretHash, &owner.ID, &owner.ProviderID,
		&owner.Subject, &owner.Issuer, &owner.Username, &owner.DisplayName, &owner.Active, &owner.LastSeenAt, &owner.CreatedAt, &owner.UpdatedAt)
	if err != nil {
		return nil, err
	}
	credential.Token = *token
	return credential, nil
}

func (r *EndgeRepository) RevokeAccessToken(ctx context.Context, id, actorID string) error {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE access_tokens SET revoked_at=COALESCE(revoked_at,NOW()),revoked_by=COALESCE(revoked_by,$2::uuid) WHERE id=$1`, id, actorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *EndgeRepository) TouchAccessToken(ctx context.Context, id string, minInterval time.Duration) error {
	_, err := r.executor(ctx).Exec(ctx, `UPDATE access_tokens SET last_used_at=NOW()
		WHERE id=$1 AND (last_used_at IS NULL OR last_used_at < NOW() - make_interval(secs => $2))`, id, minInterval.Seconds())
	return err
}
