-- name: CreateProject :one
INSERT INTO projects (
    identity,
    display_name,
    extend_settings,
    settings_id,
    navigation_id,
    folder_id,
    allowed_environment_ids,
    meta
)
VALUES (
           $1,
           $2,
           $3,
           $4,
           $5,
           $6,
           $7,
           $8
       )
    RETURNING *;

-- name: GetProjectByID :one
SELECT *
FROM projects
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetProjectByIdentity :one
SELECT *
FROM projects
WHERE identity = $1
  AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT *
FROM projects
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET
    identity = $2,
    display_name = $3,
    extend_settings = $4,
    settings_id = $5,
    navigation_id = $6,
    folder_id = $7,
    allowed_environment_ids = $8,
    meta = $9,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
    RETURNING *;

-- name: SoftDeleteProject :exec
UPDATE projects
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: RestoreProjects :exec
UPDATE projects
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: HardDeleteProject :exec
DELETE FROM projects
WHERE id = $1;

-- name: ExistsProjectByIdentity :one
SELECT EXISTS(
    SELECT 1
    FROM projects
    WHERE identity = $1
      AND deleted_at IS NULL
);

-- name: CountProject :one
SELECT COUNT(*)
FROM projects
WHERE deleted_at IS NULL;