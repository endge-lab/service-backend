-- +goose Up
-- Персональные токены доступа агентов. Секрет хранится только как SHA-256;
-- token_prefix — безопасная для показа часть, по ней владелец узнаёт токен.
CREATE TABLE access_tokens
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES service_users (id) ON DELETE CASCADE,
    workspace_id UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    description  TEXT        NOT NULL DEFAULT '' CHECK (char_length(description) <= 1000),
    scope        TEXT        NOT NULL CHECK (scope IN ('read', 'write')),
    can_commit   BOOLEAN     NOT NULL DEFAULT FALSE,
    can_delete   BOOLEAN     NOT NULL DEFAULT FALSE,
    token_prefix TEXT        NOT NULL,
    secret_hash  BYTEA       NOT NULL CHECK (octet_length(secret_hash) = 32),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    revoked_by   UUID REFERENCES service_users (id),
    CONSTRAINT access_tokens_secret_hash_unique UNIQUE (secret_hash),
    CONSTRAINT access_tokens_expiry CHECK (expires_at > created_at),
    CONSTRAINT access_tokens_flags_require_write CHECK (scope = 'write' OR (NOT can_commit AND NOT can_delete)),
    CONSTRAINT access_tokens_revocation_pair CHECK ((revoked_at IS NULL) = (revoked_by IS NULL))
);
CREATE INDEX access_tokens_user_idx ON access_tokens (user_id, created_at DESC);

-- Изменения, сделанные через токен, помечаются на уровне mutation batch:
-- actor_user_id остаётся владельцем, access_token_id показывает агента.
ALTER TABLE mutation_batches
    ADD COLUMN access_token_id UUID REFERENCES access_tokens (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE mutation_batches DROP COLUMN IF EXISTS access_token_id;
DROP TABLE IF EXISTS access_tokens;
