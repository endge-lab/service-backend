-- +goose Up
CREATE TABLE IF NOT EXISTS navigations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    project_id UUID NULL,
    tree JSONB NOT NULL DEFAULT '[]'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS navigations;
