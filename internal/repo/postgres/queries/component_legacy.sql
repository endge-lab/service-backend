-- name: CreateComponentLegacy :one
INSERT INTO components_legacy (
    workspace_id,
    project_id,
    folder_id,
    identity,
    display_name,
    description,
    component_type,
    source,
    source_format,
    props_schema,
    bindings,
    meta,
    active
)
VALUES (
           sqlc.arg(workspace_id),
           sqlc.arg(project_id),
           sqlc.arg(folder_id),
           sqlc.arg(identity),
           sqlc.arg(display_name),
           sqlc.narg(description),
           sqlc.arg(component_type),
           sqlc.arg(source),
           sqlc.arg(source_format),
           sqlc.arg(props_schema),
           sqlc.arg(bindings),
           sqlc.arg(meta),
           sqlc.arg(active)
       )
    RETURNING *;


-- name: GetComponentLegacyByID :one
SELECT *
FROM components_legacy
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;


-- name: GetComponentLegacyByIdentity :one
SELECT *
FROM components_legacy
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;


-- name: GetComponentLegacyByIdentityIncludingDeleted :one
SELECT *
FROM components_legacy
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity);


-- name: ListComponentsLegacy :many
SELECT *
FROM components_legacy
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
      OR folder_id = sqlc.narg(folder_id)
    )
  AND (
    sqlc.narg(component_type)::text IS NULL
      OR component_type = sqlc.narg(component_type)
    )
ORDER BY created_at DESC;


-- name: UpdateComponentLegacy :one
UPDATE components_legacy
SET
    folder_id = sqlc.arg(folder_id),
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    component_type = sqlc.arg(component_type),
    source = sqlc.arg(source),
    source_format = sqlc.arg(source_format),
    props_schema = sqlc.arg(props_schema),
    bindings = sqlc.arg(bindings),
    meta = sqlc.arg(meta),
    active = sqlc.arg(active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL
    RETURNING *;


-- name: SoftDeleteComponentLegacy :execrows
UPDATE components_legacy
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NULL;


-- name: RestoreComponentLegacy :execrows
UPDATE components_legacy
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND deleted_at IS NOT NULL;


-- name: HardDeleteComponentLegacy :execrows
DELETE FROM components_legacy
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);


-- name: ExistsComponentLegacyByIdentity :one
SELECT EXISTS (
    SELECT 1
    FROM components_legacy
    WHERE workspace_id = sqlc.arg(workspace_id)
      AND project_id = sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
);


-- name: CountComponentsLegacy :one
SELECT COUNT(*)
FROM components_legacy
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
      OR folder_id = sqlc.narg(folder_id)
    )
  AND (
    sqlc.narg(component_type)::text IS NULL
      OR component_type = sqlc.narg(component_type)
    );
