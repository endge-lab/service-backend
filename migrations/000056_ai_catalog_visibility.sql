-- +goose Up
ALTER TABLE ai_provider_connections
    ADD COLUMN visibility TEXT NOT NULL DEFAULT 'public'
        CHECK (visibility IN ('public', 'private')),
    ADD COLUMN owner_user_id UUID REFERENCES service_users(id) ON DELETE CASCADE;

ALTER TABLE ai_provider_connections
    ADD CONSTRAINT ai_provider_connections_visibility_owner_check CHECK (
        (visibility = 'public' AND owner_user_id IS NULL)
        OR (visibility = 'private' AND owner_user_id IS NOT NULL)
    );

ALTER TABLE ai_provider_connections
    DROP CONSTRAINT ai_provider_connections_adapter_name_key;

CREATE UNIQUE INDEX ai_provider_connections_public_name_idx
    ON ai_provider_connections (adapter, name)
    WHERE visibility = 'public';

CREATE UNIQUE INDEX ai_provider_connections_private_name_idx
    ON ai_provider_connections (owner_user_id, adapter, name)
    WHERE visibility = 'private';

ALTER TABLE ai_provider_connections
    ALTER COLUMN visibility DROP DEFAULT;

-- +goose Down
DROP INDEX IF EXISTS ai_provider_connections_private_name_idx;
DROP INDEX IF EXISTS ai_provider_connections_public_name_idx;

ALTER TABLE ai_provider_connections
    DROP CONSTRAINT IF EXISTS ai_provider_connections_visibility_owner_check,
    DROP COLUMN IF EXISTS owner_user_id,
    DROP COLUMN IF EXISTS visibility;

ALTER TABLE ai_provider_connections
    ADD CONSTRAINT ai_provider_connections_adapter_name_key UNIQUE (adapter, name);
