-- +goose Up
CREATE TABLE IF NOT EXISTS types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    is_primitive BOOLEAN NOT NULL DEFAULT FALSE,
    inherited BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    author TEXT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT types_workspace_identity_unique UNIQUE (workspace_id, identity)
);

-- +goose Down
DROP TABLE IF EXISTS types;
