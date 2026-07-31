-- name: UpsertDomainDependencyState :exec
INSERT INTO domain_dependency_states (
    workspace_id,
    owner_type,
    owner_id,
    owner_identity,
    verification_state,
    verification_error
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(owner_type),
    sqlc.arg(owner_id),
    sqlc.arg(owner_identity),
    sqlc.arg(verification_state),
    sqlc.narg(verification_error)
)
ON CONFLICT (workspace_id, owner_type, owner_id)
DO UPDATE SET
    owner_identity = EXCLUDED.owner_identity,
    verification_state = EXCLUDED.verification_state,
    verification_error = EXCLUDED.verification_error,
    updated_at = NOW();

-- name: DeleteDomainDependenciesForOwner :exec
DELETE FROM domain_dependencies
WHERE workspace_id = sqlc.arg(workspace_id)
  AND owner_type = sqlc.arg(owner_type)
  AND owner_id = sqlc.arg(owner_id);

-- name: CreateDomainDependency :exec
INSERT INTO domain_dependencies (
    workspace_id,
    owner_type,
    owner_id,
    dependency_type,
    dependency_identity,
    source_path
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(owner_type),
    sqlc.arg(owner_id),
    sqlc.arg(dependency_type),
    sqlc.arg(dependency_identity),
    sqlc.arg(source_path)
);

-- name: DeleteDomainDependencyStateForOwner :exec
DELETE FROM domain_dependency_states
WHERE workspace_id = sqlc.arg(workspace_id)
  AND owner_type = sqlc.arg(owner_type)
  AND owner_id = sqlc.arg(owner_id);

-- name: ListDomainDependencyUsages :many
SELECT
    dependency.owner_type,
    dependency.owner_id,
    state.owner_identity,
    dependency.source_path,
    state.verification_state,
    COUNT(*) OVER() AS total
FROM domain_dependencies AS dependency
JOIN domain_dependency_states AS state
  ON state.workspace_id = dependency.workspace_id
 AND state.owner_type = dependency.owner_type
 AND state.owner_id = dependency.owner_id
WHERE dependency.workspace_id = sqlc.arg(workspace_id)
  AND dependency.dependency_type = sqlc.arg(dependency_type)
  AND dependency.dependency_identity = sqlc.arg(dependency_identity)
ORDER BY dependency.owner_type, state.owner_identity, dependency.source_path
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
