-- +goose Up
CREATE TABLE IF NOT EXISTS queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    type TEXT NOT NULL,
    endpoint TEXT NULL,
    query TEXT NULL,
    sub_field TEXT NULL,
    method TEXT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    timeout_ms INTEGER NULL,
    send_as_form_urlencoded BOOLEAN NOT NULL DEFAULT FALSE,
    params JSONB NOT NULL DEFAULT '[]'::jsonb,
    return_field JSONB NULL,
    mock_data JSONB NULL,
    mock_data_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    auth JSONB NULL,
    filter_mode TEXT NULL,
    filters JSONB NOT NULL DEFAULT '[]'::jsonb,
    folder_id UUID NOT NULL REFERENCES folders(id),
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    inherited BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    author TEXT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS queries;
