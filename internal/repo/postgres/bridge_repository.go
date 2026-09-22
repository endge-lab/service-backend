package postgres

import (
	"context"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// BridgeUser проверяет текущую account/session validity без хранения access tokens в bridge.
func (r *EndgeRepository) BridgeUser(ctx context.Context, userID, sessionID string) (entities.Actor, error) {
	var actor entities.Actor
	err := r.executor(ctx).QueryRow(ctx, `
 SELECT u.id::text,u.username,u.display_name FROM service_users u
 WHERE u.id=$1 AND u.active AND ($2='' OR EXISTS (
  SELECT 1 FROM configurator_auth_sessions s WHERE s.id::text=$2
  AND s.provider_id=u.provider_id AND s.subject=u.subject AND s.issuer=u.issuer
  AND s.revoked_at IS NULL AND s.expires_at>NOW()
 ))`, userID, sessionID).Scan(&actor.ID, &actor.Username, &actor.DisplayName)
	return actor, repositoryError(err)
}
