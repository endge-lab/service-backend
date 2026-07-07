-- +goose Up
CREATE TABLE IF NOT EXISTS actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    definition JSONB NOT NULL DEFAULT '{}'::jsonb,
    input JSONB NULL,
    output JSONB NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    folder_id UUID NOT NULL REFERENCES folders(id),
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    inherited BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    author TEXT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS actions;
