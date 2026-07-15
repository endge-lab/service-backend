-- name: CreateQuery :one
INSERT INTO queries (
    project_id,
    folder_id,
    identity,
    display_name,
    description,
    query_type,
    source,
    params,
    headers,
    auth,
    timeout_ms,
    mock_data,
    mock_data_enabled,
    meta,
    active
)
VALUES (
    sqlc.arg(project_id),
    sqlc.arg(folder_id),
    sqlc.arg(identity),
    sqlc.arg(display_name),
    sqlc.narg(description),
    sqlc.arg(query_type),
    sqlc.arg(source),
    sqlc.arg(params),
    sqlc.arg(headers),
    sqlc.narg(auth),
    sqlc.narg(timeout_ms),
    sqlc.narg(mock_data),
    sqlc.arg(mock_data_enabled),
    sqlc.arg(meta),
    sqlc.arg(active)
)
RETURNING *;

-- name: GetQueryByID :one
SELECT *
FROM queries
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;

-- name: GetQueryByIDIncludingDeleted :one
SELECT *
FROM queries
WHERE id = sqlc.arg(id);

-- name: GetQueryByIdentity :one
SELECT *
FROM queries
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity)
  AND deleted_at IS NULL;

-- name: GetQueryByIdentityIncludingDeleted :one
SELECT *
FROM queries
WHERE project_id = sqlc.arg(project_id)
  AND identity = sqlc.arg(identity);

-- name: ListQueries :many
SELECT *
FROM queries
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
    OR folder_id = sqlc.narg(folder_id)
  )
  AND (
    sqlc.narg(query_type)::text IS NULL
    OR query_type = sqlc.narg(query_type)
  )
ORDER BY created_at DESC;

-- name: UpdateQuery :one
UPDATE queries
SET
    folder_id = sqlc.arg(folder_id),
    display_name = sqlc.arg(display_name),
    description = sqlc.narg(description),
    query_type = sqlc.arg(query_type),
    source = sqlc.arg(source),
    params = sqlc.arg(params),
    headers = sqlc.arg(headers),
    auth = sqlc.narg(auth),
    timeout_ms = sqlc.narg(timeout_ms),
    mock_data = sqlc.narg(mock_data),
    mock_data_enabled = sqlc.arg(mock_data_enabled),
    meta = sqlc.arg(meta),
    active = sqlc.arg(active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteQuery :execrows
UPDATE queries
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL;

-- name: RestoreQuery :execrows
UPDATE queries
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND deleted_at IS NOT NULL;

-- name: HardDeleteQuery :execrows
DELETE FROM queries
WHERE id = sqlc.arg(id);

-- name: ExistsQueryByIdentity :one
SELECT EXISTS (
    SELECT 1
    FROM queries
    WHERE project_id = sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
);

-- name: ExistsActiveQueryByIdentityOutsideProject :one
SELECT EXISTS (
    SELECT 1
    FROM queries
    WHERE project_id <> sqlc.arg(project_id)
      AND identity = sqlc.arg(identity)
      AND deleted_at IS NULL
);

-- name: CountQueries :one
SELECT COUNT(*)
FROM queries
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL
  AND (
    sqlc.narg(folder_id)::uuid IS NULL
    OR folder_id = sqlc.narg(folder_id)
  )
  AND (
    sqlc.narg(query_type)::text IS NULL
    OR query_type = sqlc.narg(query_type)
  );
