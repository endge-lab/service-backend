-- +goose Up
CREATE TABLE domain_dependency_states (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_type TEXT NOT NULL,
    owner_id UUID NOT NULL,
    owner_identity TEXT NOT NULL,
    verification_state TEXT NOT NULL,
    verification_error TEXT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, owner_type, owner_id),
    CHECK (btrim(owner_type) <> ''),
    CHECK (btrim(owner_identity) <> ''),
    CHECK (verification_state IN ('verified', 'unverified'))
);

CREATE TABLE domain_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_type TEXT NOT NULL,
    owner_id UUID NOT NULL,
    dependency_type TEXT NOT NULL,
    dependency_identity TEXT NOT NULL,
    source_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT domain_dependencies_fields_not_empty_check
    CHECK (
        btrim(owner_type) <> ''
        AND btrim(dependency_type) <> ''
        AND btrim(dependency_identity) <> ''
        AND btrim(source_path) <> ''
    ),

    CONSTRAINT domain_dependencies_owner_unique
    UNIQUE (workspace_id, owner_type, owner_id, dependency_type, dependency_identity, source_path),

    CONSTRAINT domain_dependencies_owner_state_fkey
    FOREIGN KEY (workspace_id, owner_type, owner_id)
    REFERENCES domain_dependency_states(workspace_id, owner_type, owner_id)
    ON DELETE CASCADE
);

CREATE INDEX domain_dependencies_usage_idx
    ON domain_dependencies (workspace_id, dependency_type, dependency_identity);

CREATE INDEX domain_dependencies_owner_idx
    ON domain_dependencies (workspace_id, owner_type, owner_id);

-- +goose Down
DROP TABLE IF EXISTS domain_dependencies;
DROP TABLE IF EXISTS domain_dependency_states;
