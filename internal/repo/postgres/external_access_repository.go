package postgres

import (
	"context"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
)

func (r *EndgeRepository) GetExternalAccessState(ctx context.Context, userID string) (*entities.ExternalAccessState, error) {
	value := &entities.ExternalAccessState{}
	err := r.executor(ctx).QueryRow(ctx, `SELECT config_version,token_hash,issued_at,grants_hash,last_synchronized_at
		FROM external_access_states WHERE user_id=$1`, userID).Scan(&value.ConfigVersion, &value.TokenHash, &value.IssuedAt, &value.GrantsHash, &value.LastSynchronizedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return value, err
}

func (r *EndgeRepository) HasExternalAccessHistory(ctx context.Context) (bool, error) {
	var exists bool
	err := r.executor(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM external_access_states)`).Scan(&exists)
	return exists, err
}

// ReplaceExternalAccess runs inside the actor-resolution transaction, which holds
// the service_users row lock acquired by UpsertCurrentUser. No partial set is visible.
func (r *EndgeRepository) ReplaceExternalAccess(ctx context.Context, userID string, grants []ports.AccessGrantInput, state entities.ExternalAccessState) error {
	if _, err := r.executor(ctx).Exec(ctx, `DELETE FROM workspace_memberships WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err := r.executor(ctx).Exec(ctx, `DELETE FROM access_grants WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, grant := range grants {
		if _, _, err := r.UpsertAccessGrant(ctx, grant); err != nil {
			return err
		}
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO external_access_states(user_id,config_version,token_hash,issued_at,grants_hash)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id) DO UPDATE SET config_version=EXCLUDED.config_version,
		token_hash=EXCLUDED.token_hash,issued_at=EXCLUDED.issued_at,grants_hash=EXCLUDED.grants_hash,last_synchronized_at=NOW()`,
		userID, state.ConfigVersion, state.TokenHash, state.IssuedAt, state.GrantsHash)
	return err
}

// Local changes invalidate idempotence without discarding the token watermark.
func (r *EndgeRepository) invalidateExternalAccess(ctx context.Context, userID string) error {
	_, err := r.executor(ctx).Exec(ctx, `UPDATE external_access_states SET config_version='' WHERE user_id=$1 AND config_version<>''`, userID)
	return err
}
