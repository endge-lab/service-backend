-- +goose Up
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    configuration JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT workspaces_identity_not_empty_check
        CHECK (btrim(identity) <> ''),

    CONSTRAINT workspaces_display_name_not_empty_check
        CHECK (btrim(display_name) <> ''),

    CONSTRAINT workspaces_configuration_object_check
        CHECK (jsonb_typeof(configuration) = 'object')
);

-- +goose Down
DROP TABLE IF EXISTS workspaces;
