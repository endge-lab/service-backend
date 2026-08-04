-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE service_users
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    provider_id  TEXT        NOT NULL,
    subject      TEXT        NOT NULL,
    issuer       TEXT        NOT NULL,
    username     TEXT        NOT NULL DEFAULT '',
    display_name TEXT        NOT NULL DEFAULT '',
    active       BOOLEAN     NOT NULL DEFAULT TRUE,
    is_system    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT service_users_provider_subject_key UNIQUE (provider_id, subject)
);

-- +goose Down
DROP TABLE IF EXISTS service_users;
