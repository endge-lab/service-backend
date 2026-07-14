-- +goose Up
CREATE TABLE IF NOT EXISTS folders (
                                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NULL,
    entity_type TEXT NOT NULL,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT folders_entity_type_check
    CHECK (entity_type IN ('components', 'converters', 'queries', 'data-views')),

    CONSTRAINT folders_identity_not_empty_check
    CHECK (btrim(identity) <> ''),

    CONSTRAINT folders_display_name_not_empty_check
    CHECK (btrim(display_name) <> ''),

    CONSTRAINT folders_not_self_parent_check
    CHECK (parent_id IS NULL OR parent_id <> id),

    CONSTRAINT folders_root_has_no_parent_check
    CHECK (NOT is_root OR parent_id IS NULL)
    );

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_identity_unique
    ON folders (project_id, entity_type, identity)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_identity_unique
    ON folders (entity_type, identity)
    WHERE project_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_root_unique
    ON folders (project_id, entity_type)
    WHERE project_id IS NOT NULL AND is_root;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_root_unique
    ON folders (entity_type)
    WHERE project_id IS NULL AND is_root;

-- +goose Down
DROP TABLE IF EXISTS folders;