-- +goose Up
CREATE TABLE IF NOT EXISTS settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    project_id UUID NULL,
    environment TEXT NULL,
    vars JSONB NOT NULL DEFAULT '[]'::jsonb,
    auth JSONB NULL,
    vocabs JSONB NOT NULL DEFAULT '[]'::jsonb,
    sse JSONB NULL,
    updates JSONB NOT NULL DEFAULT '[]'::jsonb,
    custom_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS settings;
