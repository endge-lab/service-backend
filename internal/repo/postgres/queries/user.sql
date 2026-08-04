-- name: UpsertCurrentUser :one
INSERT INTO service_users (provider_id, subject, issuer, username, display_name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider_id, subject) DO UPDATE SET
  issuer = EXCLUDED.issuer,
  username = CASE WHEN EXCLUDED.username <> '' THEN EXCLUDED.username ELSE service_users.username END,
  display_name = CASE WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name ELSE service_users.display_name END,
  updated_at = NOW(),
  last_seen_at = NOW()
RETURNING id, provider_id, subject, issuer, username, display_name, active, created_at, updated_at, last_seen_at;
