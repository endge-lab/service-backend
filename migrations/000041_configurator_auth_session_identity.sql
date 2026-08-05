-- +goose Up
ALTER TABLE configurator_auth_sessions
    ADD COLUMN IF NOT EXISTS identity_refresh_at TIMESTAMPTZ;

-- Старые access token больше не хранятся. Существующие сессии должны пройти
-- identity refresh при следующем обращении либо быть безопасно отозваны.
UPDATE configurator_auth_sessions
SET identity_refresh_at = NOW()
WHERE identity_refresh_at IS NULL;

ALTER TABLE configurator_auth_sessions
    ALTER COLUMN identity_refresh_at SET NOT NULL,
    DROP COLUMN IF EXISTS access_token_encrypted,
    DROP COLUMN IF EXISTS access_expires_at;

DROP INDEX IF EXISTS configurator_auth_sessions_expires_at_idx;
CREATE INDEX configurator_auth_sessions_expires_at_idx
    ON configurator_auth_sessions (expires_at);

CREATE INDEX IF NOT EXISTS configurator_auth_sessions_revoked_at_idx
    ON configurator_auth_sessions (revoked_at)
    WHERE revoked_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS configurator_auth_sessions_revoked_at_idx;
DROP INDEX IF EXISTS configurator_auth_sessions_expires_at_idx;
CREATE INDEX configurator_auth_sessions_expires_at_idx
    ON configurator_auth_sessions (expires_at)
    WHERE revoked_at IS NULL;

ALTER TABLE configurator_auth_sessions
    ADD COLUMN IF NOT EXISTS access_token_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS access_expires_at TIMESTAMPTZ,
    DROP COLUMN IF EXISTS identity_refresh_at;
