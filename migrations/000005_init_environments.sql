-- +goose Up
CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    folder_id UUID NULL,
    configuration JSONB NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT environments_workspace_identity_unique UNIQUE (workspace_id, identity),
    CONSTRAINT environments_identity_not_empty_check CHECK (btrim(identity) <> ''),
    CONSTRAINT environments_display_name_not_empty_check CHECK (btrim(display_name) <> ''),
    CONSTRAINT environments_configuration_object_check CHECK (jsonb_typeof(configuration) = 'object')
);

-- +goose Down
DROP TABLE IF EXISTS environments;
