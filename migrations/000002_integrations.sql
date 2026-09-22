-- +goose Up
CREATE TABLE integrations
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    identity           TEXT        NOT NULL UNIQUE CHECK (char_length(identity) BETWEEN 1 AND 160),
    display_name       TEXT        NOT NULL,
    description        TEXT,
    version            TEXT        NOT NULL,
    managed_by         TEXT        NOT NULL DEFAULT 'user' CHECK (managed_by IN ('user', 'system', 'integration')),
    managed_by_id      TEXT,
    meta               JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active             BOOLEAN     NOT NULL DEFAULT TRUE,
    deleted_at         TIMESTAMPTZ,
    created_by UUID        NOT NULL REFERENCES service_users (id),
    updated_by UUID        NOT NULL REFERENCES service_users (id),
    revision           INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS integrations;
