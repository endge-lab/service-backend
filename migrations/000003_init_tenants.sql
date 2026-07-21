-- +goose Up
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    description TEXT NULL,
    folder_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tenants_workspace_identity_unique UNIQUE (workspace_id, identity)
);

-- +goose Down
DROP TABLE IF EXISTS tenants;
