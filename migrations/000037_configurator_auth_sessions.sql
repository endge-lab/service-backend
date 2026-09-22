-- +goose Up
CREATE TABLE configurator_auth_transactions
(
    state_hash         BYTEA PRIMARY KEY,
    browser_nonce_hash BYTEA       NOT NULL,
    verifier_encrypted BYTEA       NOT NULL,
    oidc_nonce_encrypted BYTEA      NOT NULL,
    return_url         TEXT        NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX configurator_auth_transactions_expires_at_idx
    ON configurator_auth_transactions (expires_at);

CREATE TABLE configurator_auth_sessions
(
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash              BYTEA       NOT NULL UNIQUE,
    provider_id             TEXT        NOT NULL,
    subject                 TEXT        NOT NULL,
    issuer                  TEXT        NOT NULL,
    username                TEXT        NOT NULL DEFAULT '',
    display_name            TEXT        NOT NULL DEFAULT '',
    groups_json             JSONB       NOT NULL DEFAULT '[]'::jsonb,
    platform_admin          BOOLEAN     NOT NULL DEFAULT FALSE,
    access_token_encrypted  BYTEA       NOT NULL,
    refresh_token_encrypted BYTEA,
    access_expires_at       TIMESTAMPTZ NOT NULL,
    expires_at              TIMESTAMPTZ NOT NULL,
    revoked_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX configurator_auth_sessions_expires_at_idx
    ON configurator_auth_sessions (expires_at)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS configurator_auth_sessions;
DROP TABLE IF EXISTS configurator_auth_transactions;
