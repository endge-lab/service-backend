-- name: CreateFolder :one
INSERT INTO folders (
    workspace_id,
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
    sqlc.arg(workspace_id),
    sqlc.narg(project_id),
    sqlc.arg(entity_type),
    sqlc.arg(identity),
    sqlc.arg(display_name),
    sqlc.narg(description),
    sqlc.narg(parent_id),
    sqlc.arg(is_root),
    sqlc.arg(is_system),
    sqlc.arg(meta)
)
RETURNING *;

-- name: GetFolderByID :one
SELECT *
FROM folders
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;

-- name: GetFolderByIDIncludingDeleted :one
SELECT *
FROM folders
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);

-- name: GetFolderByProjectEntityIdentity :one
SELECT *
FROM folders
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)
  AND entity_type = sqlc.arg(entity_type)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;

-- name: GetFolderByProjectEntityIdentityIncludingDeleted :one
SELECT *
FROM folders
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)
  AND entity_type = sqlc.arg(entity_type)
  AND identity = sqlc.arg(identity);

-- name: ListFoldersByProjectAndEntityType :many
SELECT *
FROM folders
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)
  AND entity_type = sqlc.arg(entity_type)
  AND deleted_at IS NULL
ORDER BY is_root DESC, display_name ASC, created_at ASC;

-- name: UpdateFolder :one
UPDATE folders
SET
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    parent_id = sqlc.narg(parent_id),
    meta = sqlc.arg(meta),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteFolder :execrows
UPDATE folders
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;

-- name: RestoreFolder :execrows
UPDATE folders
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NOT NULL;

-- name: HardDeleteFolder :execrows
DELETE FROM folders
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);

-- name: ExistsFolderByProjectEntityIdentity :one
SELECT EXISTS(
    SELECT 1
    FROM folders
    WHERE workspace_id = sqlc.arg(workspace_id)
      AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)
      AND entity_type = sqlc.arg(entity_type)
      AND identity = sqlc.arg(identity)
);

-- name: CountFoldersByProjectAndEntityType :one
SELECT COUNT(*)
FROM folders
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)
  AND entity_type = sqlc.arg(entity_type)
  AND deleted_at IS NULL;
