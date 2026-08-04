-- +goose Up
CREATE TABLE workspaces
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    identity           TEXT        NOT NULL UNIQUE CHECK (char_length(identity) BETWEEN 1 AND 160),
    display_name       TEXT        NOT NULL,
    description        TEXT,
    data_mode          TEXT        NOT NULL DEFAULT 'development',
    configuration      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    meta               JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active             BOOLEAN     NOT NULL DEFAULT TRUE,
    generation         UUID        NOT NULL DEFAULT gen_random_uuid(),
    created_by UUID        NOT NULL REFERENCES service_users (id),
    updated_by UUID        NOT NULL REFERENCES service_users (id),
    head_sequence      BIGINT      NOT NULL DEFAULT 0,
    revision           INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS workspaces;
