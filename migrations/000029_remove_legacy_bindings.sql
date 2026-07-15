-- +goose Up
DROP TABLE IF EXISTS presentation_bindings;
DROP TABLE IF EXISTS behavior_bindings;

-- +goose Down
CREATE TABLE IF NOT EXISTS behavior_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    owner_type TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    script_ref TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'replace',
    priority INTEGER NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    environment_id UUID NULL REFERENCES environments(id) ON DELETE SET NULL,
    is_inherited BOOLEAN NOT NULL DEFAULT FALSE,
    origin_binding_id UUID NULL REFERENCES behavior_bindings(id) ON DELETE SET NULL,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS presentation_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    owner_type TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NULL,
    role TEXT NOT NULL,
    renderer_ref TEXT NOT NULL,
    when_expression TEXT NULL,
    mode TEXT NOT NULL DEFAULT 'replace',
    priority INTEGER NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    environment_id UUID NULL REFERENCES environments(id) ON DELETE SET NULL,
    is_inherited BOOLEAN NOT NULL DEFAULT FALSE,
    origin_binding_id UUID NULL REFERENCES presentation_bindings(id) ON DELETE SET NULL,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
