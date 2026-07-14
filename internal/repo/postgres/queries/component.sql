-- name: CreateComponent :one
INSERT INTO components (
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


-- name: GetComponentByID :one
SELECT *
FROM components
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;


-- name: GetComponentByIdentity :one
SELECT *
FROM components
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;


-- name: GetComponentByIdentityIncludingDeleted :one
SELECT *
FROM components
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity);


-- name: ListComponents :many
SELECT *
FROM components
WHERE project_id = sqlc.arg(project_id)
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


-- name: UpdateComponent :one
UPDATE components
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
  AND deleted_at IS NULL
    RETURNING *;


-- name: SoftDeleteComponent :execrows
UPDATE components
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;


-- name: RestoreComponent :execrows
UPDATE components
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NOT NULL;


-- name: HardDeleteComponent :execrows
DELETE FROM components
WHERE id = sqlc.arg(id);


-- name: ExistsComponentByIdentity :one
SELECT EXISTS (
    SELECT 1
    FROM components
    WHERE project_id = sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
);


-- name: CountComponents :one
SELECT COUNT(*)
FROM components
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
      OR folder_id = sqlc.narg(folder_id)
    )
  AND (
    sqlc.narg(component_type)::text IS NULL
      OR component_type = sqlc.narg(component_type)
    );