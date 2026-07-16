-- +goose Up
CREATE TABLE IF NOT EXISTS converters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    converter_type TEXT NOT NULL,
    source JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT converters_project_identity_unique
    UNIQUE (project_id, identity),

    CONSTRAINT converters_identity_not_empty_check
    CHECK (btrim(identity) <> ''),

    CONSTRAINT converters_display_name_not_empty_check
    CHECK (btrim(display_name) <> ''),

    CONSTRAINT converters_type_not_empty_check
    CHECK (btrim(converter_type) <> '')
    );

-- +goose Down
DROP TABLE IF EXISTS converters;