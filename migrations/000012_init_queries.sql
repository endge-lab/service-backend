-- +goose Up
CREATE TABLE IF NOT EXISTS queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    query_type TEXT NOT NULL,
    source JSONB NOT NULL DEFAULT '{}'::jsonb,
    params JSONB NOT NULL DEFAULT '[]'::jsonb,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    auth JSONB NULL,
    timeout_ms INTEGER NULL,
    mock_data JSONB NULL,
    mock_data_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT queries_project_identity_unique
        UNIQUE (project_id, identity),

    CONSTRAINT queries_identity_not_empty_check
        CHECK (btrim(identity) <> ''),

    CONSTRAINT queries_display_name_not_empty_check
        CHECK (btrim(display_name) <> ''),

    CONSTRAINT queries_type_not_empty_check
        CHECK (btrim(query_type) <> ''),

    CONSTRAINT queries_source_object_check
        CHECK (jsonb_typeof(source) = 'object'),

    CONSTRAINT queries_params_array_check
        CHECK (jsonb_typeof(params) = 'array'),

    CONSTRAINT queries_headers_object_check
        CHECK (jsonb_typeof(headers) = 'object'),

    CONSTRAINT queries_timeout_positive_check
        CHECK (timeout_ms IS NULL OR timeout_ms > 0)
);

-- +goose Down
DROP TABLE IF EXISTS queries;
