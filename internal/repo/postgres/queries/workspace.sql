-- name: CreateWorkspace :one
INSERT INTO workspaces (
    identity,
    display_name,
    configuration
)
VALUES (
    sqlc.arg(identity),
    sqlc.arg(display_name),
    sqlc.arg(configuration)
)
RETURNING *;

-- name: ListWorkspaces :many
SELECT *
FROM workspaces
ORDER BY created_at DESC;

-- name: GetWorkspaceByIdentity :one
SELECT *
FROM workspaces
WHERE identity = sqlc.arg(identity);

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    display_name = sqlc.arg(display_name),
    configuration = sqlc.arg(configuration),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
