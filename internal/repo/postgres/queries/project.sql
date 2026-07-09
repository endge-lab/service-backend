-- name: CreateProject :one
INSERT INTO projects (
    identity,
    display_name,
    description,
    active,
    meta
)
VALUES (
           $1,
           $2,
           $3,
           $4,
           $5
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

-- name: GetProjectByIdentityIncludingDeleted :one
SELECT *
FROM projects
WHERE identity = $1;

-- name: ListProjects :many
SELECT *
FROM projects
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET
    display_name = $2,
    description = $3,
    active = $4,
    meta = $5,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
    RETURNING *;

-- name: SoftDeleteProject :execrows
UPDATE projects
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: RestoreProject :execrows
UPDATE projects
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NOT NULL;

-- name: HardDeleteProject :execrows
DELETE FROM projects
WHERE id = $1;

-- name: ExistsProjectByIdentity :one
SELECT EXISTS(
    SELECT 1
    FROM projects
    WHERE identity = $1
);

-- name: CountProject :one
SELECT COUNT(*)
FROM projects
WHERE deleted_at IS NULL;
