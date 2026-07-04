-- name: UpsertServiceUserFromIdentity :one
INSERT INTO service_users (
  id,
  auth_user_id,
  username,
  display_name,
  role,
  created_at,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (auth_user_id) DO UPDATE
SET
  username = CASE
    WHEN EXCLUDED.username <> '' THEN EXCLUDED.username
    ELSE service_users.username
  END,
  display_name = CASE
    WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name
    ELSE service_users.display_name
  END,
  role = CASE
    WHEN EXCLUDED.role <> '' THEN EXCLUDED.role
    ELSE service_users.role
  END,
  updated_at = NOW()
RETURNING id, auth_user_id, username, display_name, role, created_at, updated_at;
