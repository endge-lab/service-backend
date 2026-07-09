-- name: CreateFolder :one
INSERT INTO folders (
    project_id,
    entity_type,
    identity,
    display_name,
    description,
    parent_id,
    is_root,
    is_system,
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
    $8,
    $9
)
RETURNING *;

-- name: GetFolderByID :one
SELECT *
FROM folders
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetFolderByIDIncludingDeleted :one
SELECT *
FROM folders
WHERE id = $1;

-- name: GetFolderByProjectEntityIdentity :one
SELECT *
FROM folders
WHERE project_id IS NOT DISTINCT FROM $1
  AND entity_type = $2
  AND identity = $3
  AND deleted_at IS NULL;

-- name: GetFolderByProjectEntityIdentityIncludingDeleted :one
SELECT *
FROM folders
WHERE project_id IS NOT DISTINCT FROM $1
  AND entity_type = $2
  AND identity = $3;

-- name: ListFoldersByProjectAndEntityType :many
SELECT *
FROM folders
WHERE project_id IS NOT DISTINCT FROM $1
  AND entity_type = $2
  AND deleted_at IS NULL
ORDER BY is_root DESC, display_name ASC, created_at ASC;

-- name: UpdateFolder :one
UPDATE folders
SET
    display_name = $2,
    description = $3,
    parent_id = $4,
    meta = $5,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteFolder :execrows
UPDATE folders
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: RestoreFolder :execrows
UPDATE folders
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NOT NULL;

-- name: HardDeleteFolder :execrows
DELETE FROM folders
WHERE id = $1;

-- name: ExistsFolderByProjectEntityIdentity :one
SELECT EXISTS(
    SELECT 1
    FROM folders
    WHERE project_id IS NOT DISTINCT FROM $1
      AND entity_type = $2
      AND identity = $3
);

-- name: CountFoldersByProjectAndEntityType :one
SELECT COUNT(*)
FROM folders
WHERE project_id IS NOT DISTINCT FROM $1
  AND entity_type = $2
  AND deleted_at IS NULL;
