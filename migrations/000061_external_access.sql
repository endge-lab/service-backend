-- +goose Up
-- Derived assignments remain in access_grants; only the synchronization watermark lives here.
CREATE TABLE external_access_states (
    user_id UUID PRIMARY KEY REFERENCES service_users(id) ON DELETE CASCADE,
    config_version TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    grants_hash TEXT NOT NULL,
    last_synchronized_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A session stores the derived mapping, never the access token or its raw claims.
ALTER TABLE configurator_auth_sessions ADD COLUMN external_access JSONB;

-- +goose Down
ALTER TABLE configurator_auth_sessions DROP COLUMN external_access;
DROP TABLE external_access_states;
