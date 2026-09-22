-- +goose Up
CREATE TABLE ai_provider_connections (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    adapter TEXT NOT NULL CHECK (adapter IN ('anthropic', 'ollama')),
    base_url TEXT NOT NULL DEFAULT '',
    credential_encrypted BYTEA,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES service_users(id),
    updated_by UUID NOT NULL REFERENCES service_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (adapter, name)
);

CREATE TABLE ai_model_profiles (
    id UUID PRIMARY KEY,
    connection_id UUID NOT NULL REFERENCES ai_provider_connections(id) ON DELETE CASCADE,
    provider_model_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES service_users(id),
    updated_by UUID NOT NULL REFERENCES service_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (connection_id, provider_model_id)
);

CREATE UNIQUE INDEX ai_model_profiles_one_enabled_default_idx
    ON ai_model_profiles ((is_default))
    WHERE is_default AND enabled;

-- +goose Down
DROP TABLE IF EXISTS ai_model_profiles;
DROP TABLE IF EXISTS ai_provider_connections;
