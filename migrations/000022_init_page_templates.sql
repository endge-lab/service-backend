-- +goose Up
CREATE TABLE IF NOT EXISTS page_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    inherited BOOLEAN NOT NULL DEFAULT FALSE,
    areas JSONB NOT NULL DEFAULT '[]'::jsonb,
    preview JSONB NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS page_templates;
