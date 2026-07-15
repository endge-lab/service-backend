-- +goose Up
CREATE TABLE IF NOT EXISTS data_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
    query_id UUID NOT NULL REFERENCES queries(id) ON DELETE CASCADE,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    view_type TEXT NOT NULL,
    source JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT data_views_project_identity_unique
        UNIQUE (project_id, identity),

    CONSTRAINT data_views_identity_not_empty_check
        CHECK (btrim(identity) <> ''),

    CONSTRAINT data_views_display_name_not_empty_check
        CHECK (btrim(display_name) <> ''),

    CONSTRAINT data_views_type_not_empty_check
        CHECK (btrim(view_type) <> ''),

    CONSTRAINT data_views_source_object_check
        CHECK (jsonb_typeof(source) = 'object'),

    CONSTRAINT data_views_input_schema_object_check
        CHECK (jsonb_typeof(input_schema) = 'object'),

    CONSTRAINT data_views_output_schema_object_check
        CHECK (jsonb_typeof(output_schema) = 'object')
);

-- +goose Down
DROP TABLE IF EXISTS data_views;
