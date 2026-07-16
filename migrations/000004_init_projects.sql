-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT projects_identity_unique UNIQUE (identity),
    CONSTRAINT projects_identity_not_empty_check CHECK (btrim(identity) <> ''),
    CONSTRAINT projects_display_name_not_empty_check CHECK (btrim(display_name) <> '')
);

-- +goose Down
DROP TABLE IF EXISTS projects;
