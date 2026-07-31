-- name: CreateTenant :one
INSERT INTO tenants (
    workspace_id,
    identity,
    display_name,
    code,
    description,
    folder_id,
    configuration
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(identity),
    sqlc.arg(display_name),
    sqlc.arg(code),
    sqlc.narg(description),
    sqlc.narg(folder_id),
    sqlc.arg(configuration)
)
RETURNING *;

-- name: ListTenants :many
SELECT *
FROM tenants
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
    OR folder_id = sqlc.narg(folder_id)
  )
ORDER BY created_at DESC;

-- name: GetTenantByIdentity :one
SELECT *
FROM tenants
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity = sqlc.arg(identity);

-- name: UpdateTenant :one
UPDATE tenants
SET
    display_name = sqlc.arg(display_name),
    code = sqlc.arg(code),
    description = sqlc.narg(description),
    folder_id = sqlc.narg(folder_id),
    configuration = sqlc.arg(configuration),
    updated_at = NOW()
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity = sqlc.arg(identity)
RETURNING *;

-- name: HardDeleteTenant :execrows
DELETE FROM tenants
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity = sqlc.arg(identity);

-- name: GetTenantFolderByID :one
SELECT id
FROM folders
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND entity_type = 'tenants'
  AND deleted_at IS NULL;

-- name: GetTenantRootFolder :one
SELECT id
FROM folders
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id IS NULL
  AND entity_type = 'tenants'
  AND identity = 'root-tenants'
  AND is_root = TRUE
  AND deleted_at IS NULL;
