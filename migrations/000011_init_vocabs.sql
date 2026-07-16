-- +goose Up
CREATE TABLE IF NOT EXISTS vocabs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    mode TEXT NOT NULL,
    base_api_url TEXT NULL,
    collection_slug TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS vocabs;
