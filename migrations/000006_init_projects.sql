-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    extend_settings BOOLEAN NOT NULL DEFAULT FALSE,
    settings_id UUID NULL REFERENCES settings(id) ON DELETE SET NULL,
    navigation_id UUID NULL REFERENCES navigations(id) ON DELETE SET NULL,
    folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    allowed_environment_ids UUID[] NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE folders
    ADD CONSTRAINT folders_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

ALTER TABLE settings
    ADD CONSTRAINT settings_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

ALTER TABLE navigations
    ADD CONSTRAINT navigations_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE IF EXISTS navigations
    DROP CONSTRAINT IF EXISTS navigations_project_id_fkey;

ALTER TABLE IF EXISTS settings
    DROP CONSTRAINT IF EXISTS settings_project_id_fkey;

ALTER TABLE IF EXISTS folders
    DROP CONSTRAINT IF EXISTS folders_project_id_fkey;

DROP TABLE IF EXISTS projects;
