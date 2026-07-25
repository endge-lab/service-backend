-- name: CreateProject :one
INSERT INTO projects (
    workspace_id,
    identity,
    display_name,
    description,
    active,
    meta
)
VALUES (
           sqlc.arg(workspace_id),
           sqlc.arg(identity),
           sqlc.arg(display_name),
           sqlc.narg(description),
           sqlc.arg(active),
           sqlc.arg(meta)
       )
    RETURNING *;

-- name: GetProjectByID :one
SELECT *
FROM projects
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;

-- name: GetProjectByIdentity :one
SELECT *
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;

-- name: GetProjectByIdentityIncludingDeleted :one
SELECT *
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity = sqlc.arg(identity);

-- name: ListProjects :many
SELECT *
FROM projects
WHERE deleted_at IS NULL
  AND workspace_id = sqlc.arg(workspace_id)
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    active = sqlc.arg(active),
    meta = sqlc.arg(meta),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL
    RETURNING *;

-- name: SoftDeleteProject :execrows
UPDATE projects
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;

-- name: RestoreProject :execrows
UPDATE projects
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NOT NULL;

-- name: HardDeleteProject :execrows
DELETE FROM projects
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);

-- name: ExistsProjectByIdentity :one
SELECT EXISTS(
    SELECT 1
    FROM projects
    WHERE workspace_id = sqlc.arg(workspace_id)
      AND identity = sqlc.arg(identity)
);

-- name: CountProject :one
SELECT COUNT(*)
FROM projects
WHERE deleted_at IS NULL
  AND workspace_id = sqlc.arg(workspace_id);
