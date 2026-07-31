-- +goose Up
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT NULL,
    folder_id UUID NULL,
    configuration JSONB NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tenants_workspace_identity_unique UNIQUE (workspace_id, identity),
    CONSTRAINT tenants_workspace_code_unique UNIQUE (workspace_id, code),
    CONSTRAINT tenants_identity_not_empty_check CHECK (btrim(identity) <> ''),
    CONSTRAINT tenants_display_name_not_empty_check CHECK (btrim(display_name) <> ''),
    CONSTRAINT tenants_code_not_empty_check CHECK (btrim(code) <> ''),
    CONSTRAINT tenants_configuration_object_check CHECK (jsonb_typeof(configuration) = 'object')
);

-- +goose Down
DROP TABLE IF EXISTS tenants;
