-- +goose Up
CREATE TABLE build_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    display_name VARCHAR(160) NOT NULL,
    visibility VARCHAR(16) NOT NULL CHECK (visibility IN ('shared', 'private')),
    owner_user_id UUID NOT NULL REFERENCES service_users(id),
    settings_version INTEGER NOT NULL DEFAULT 1 CHECK (settings_version = 1),
    settings JSONB NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_by UUID NOT NULL REFERENCES service_users(id),
    updated_by UUID NOT NULL REFERENCES service_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, identity)
);

CREATE INDEX idx_build_profiles_workspace_visibility
    ON build_profiles(workspace_id, visibility, owner_user_id);

-- +goose Down
DROP TABLE IF EXISTS build_profiles;
