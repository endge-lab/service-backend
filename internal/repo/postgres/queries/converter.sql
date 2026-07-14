-- name: CreateConverter :one
INSERT INTO converters (
    project_id,
    folder_id,
    identity,
    display_name,
    description,
    converter_type,
    source,
    is_system,
    meta,
    active
)
VALUES (
           sqlc.arg(project_id),
           sqlc.arg(folder_id),
           sqlc.arg(identity),
           sqlc.arg(display_name),
           sqlc.narg(description),
           sqlc.arg(converter_type),
           sqlc.arg(source),
           sqlc.arg(is_system),
           sqlc.arg(meta),
           sqlc.arg(active)
       )
    RETURNING *;


-- name: GetConverterByID :one
SELECT *
FROM converters
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;


-- name: GetConverterByIdentity :one
SELECT *
FROM converters
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;


-- name: GetConverterByIdentityIncludingDeleted :one
SELECT *
FROM converters
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity);


-- name: ListConverters :many
SELECT *
FROM converters
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
      OR folder_id = sqlc.narg(folder_id)
    )
ORDER BY created_at DESC;


-- name: UpdateConverter :one
UPDATE converters
SET
    folder_id = sqlc.arg(folder_id),
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    converter_type = sqlc.arg(converter_type),
    source = sqlc.arg(source),
    is_system = sqlc.arg(is_system),
    meta = sqlc.arg(meta),
    active = sqlc.arg(active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
    RETURNING *;


-- name: SoftDeleteConverter :execrows
UPDATE converters
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;


-- name: RestoreConverter :execrows
UPDATE converters
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NOT NULL;


-- name: HardDeleteConverter :execrows
DELETE FROM converters
WHERE id = sqlc.arg(id);


-- name: ExistsConverterByIdentity :one
SELECT EXISTS (
    SELECT 1
    FROM converters
    WHERE project_id = sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
);


-- name: CountConverters :one
SELECT COUNT(*)
FROM converters
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
      OR folder_id = sqlc.narg(folder_id)
    );