-- +goose Up
CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    component_type TEXT NOT NULL,
    input_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    jsx_script TEXT NULL,
    row_size TEXT NULL,
    bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    columns JSONB NOT NULL DEFAULT '[]'::jsonb,
    schema JSONB NOT NULL DEFAULT '{}'::jsonb,
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
DROP TABLE IF EXISTS components;
