-- +goose Up
CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    component_type TEXT NOT NULL,
    source TEXT NOT NULL,
    source_format TEXT NOT NULL DEFAULT 'sfc',
    props_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT components_project_identity_unique
    UNIQUE (project_id, identity),

    CONSTRAINT components_identity_not_empty_check
    CHECK (btrim(identity) <> ''),

    CONSTRAINT components_display_name_not_empty_check
    CHECK (btrim(display_name) <> ''),

    CONSTRAINT components_source_not_empty_check
    CHECK (btrim(source) <> ''),

    CONSTRAINT components_type_check
    CHECK (component_type = 'component-sfc'),

    CONSTRAINT components_source_format_check
    CHECK (source_format = 'sfc')
    );

-- +goose Down
DROP TABLE IF EXISTS components;