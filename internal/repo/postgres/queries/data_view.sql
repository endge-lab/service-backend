-- name: CreateDataView :one
INSERT INTO data_views (
    project_id,
    folder_id,
    query_id,
    identity,
    display_name,
    description,
    view_type,
    source,
    input_schema,
    output_schema,
    meta,
    active
)
VALUES (
    sqlc.arg(project_id),
    sqlc.arg(folder_id),
    sqlc.arg(query_id),
    sqlc.arg(identity),
    sqlc.arg(display_name),
    sqlc.narg(description),
    sqlc.arg(view_type),
    sqlc.arg(source),
    sqlc.arg(input_schema),
    sqlc.arg(output_schema),
    sqlc.arg(meta),
    sqlc.arg(active)
)
RETURNING *;

-- name: GetDataViewByID :one
SELECT *
FROM data_views
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;

-- name: GetDataViewByIdentity :one
SELECT *
FROM data_views
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;

-- name: GetDataViewByIdentityIncludingDeleted :one
SELECT *
FROM data_views
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity);

-- name: ListDataViews :many
SELECT *
FROM data_views
WHERE data_views.project_id = sqlc.arg(project_id)
  AND data_views.deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
    OR data_views.folder_id = sqlc.narg(folder_id)
  )
  AND (
    sqlc.narg(query_id)::uuid IS NULL
    OR data_views.query_id = sqlc.narg(query_id)
  )
  AND EXISTS (
    SELECT 1
    FROM queries
    WHERE queries.id = data_views.query_id
      AND queries.deleted_at IS NULL
  )
ORDER BY data_views.created_at DESC;

-- name: UpdateDataView :one
UPDATE data_views
SET
    folder_id = sqlc.arg(folder_id),
    query_id = sqlc.arg(query_id),
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    view_type = sqlc.arg(view_type),
    source = sqlc.arg(source),
    input_schema = sqlc.arg(input_schema),
    output_schema = sqlc.arg(output_schema),
    meta = sqlc.arg(meta),
    active = sqlc.arg(active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDataView :execrows
UPDATE data_views
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;

-- name: RestoreDataView :execrows
UPDATE data_views
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NOT NULL;

-- name: HardDeleteDataView :execrows
DELETE FROM data_views
WHERE id = sqlc.arg(id);

-- name: ExistsDataViewByIdentity :one
SELECT EXISTS (
    SELECT 1
    FROM data_views
    WHERE project_id = sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
);

-- name: CountDataViews :one
SELECT COUNT(*)
FROM data_views
WHERE data_views.project_id = sqlc.arg(project_id)
  AND data_views.deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
    OR data_views.folder_id = sqlc.narg(folder_id)
  )
  AND (
    sqlc.narg(query_id)::uuid IS NULL
    OR data_views.query_id = sqlc.narg(query_id)
  )
  AND EXISTS (
    SELECT 1
    FROM queries
    WHERE queries.id = data_views.query_id
      AND queries.deleted_at IS NULL
  );
