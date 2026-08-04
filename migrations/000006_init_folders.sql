-- +goose Up
CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NULL,
    entity_type TEXT NOT NULL,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT folders_workspace_id_unique UNIQUE (workspace_id, id),

    CONSTRAINT folders_workspace_project_fkey
    FOREIGN KEY (workspace_id, project_id)
    REFERENCES projects(workspace_id, id)
    ON DELETE CASCADE,

    CONSTRAINT folders_entity_type_check
    CHECK (entity_type IN ('components-legacy', 'converters', 'queries', 'data-views', 'tenants', 'environments')),

    CONSTRAINT folders_identity_not_empty_check
    CHECK (btrim(identity) <> ''),

    CONSTRAINT folders_display_name_not_empty_check
    CHECK (btrim(display_name) <> ''),

    CONSTRAINT folders_not_self_parent_check
    CHECK (parent_id IS NULL OR parent_id <> id),

    CONSTRAINT folders_root_has_no_parent_check
    CHECK (NOT is_root OR parent_id IS NULL)
    );

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_identity_unique
    ON folders (workspace_id, project_id, entity_type, identity)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_identity_unique
    ON folders (workspace_id, entity_type, identity)
    WHERE project_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_root_unique
    ON folders (workspace_id, project_id, entity_type)
    WHERE project_id IS NOT NULL AND is_root;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_root_unique
    ON folders (workspace_id, entity_type)
    WHERE project_id IS NULL AND is_root;

ALTER TABLE tenants
    ADD CONSTRAINT tenants_folder_id_fkey
    FOREIGN KEY (workspace_id, folder_id)
    REFERENCES folders(workspace_id, id)
    ON DELETE SET NULL (folder_id);

ALTER TABLE environments
    ADD CONSTRAINT environments_folder_id_fkey
    FOREIGN KEY (workspace_id, folder_id)
    REFERENCES folders(workspace_id, id)
    ON DELETE SET NULL (folder_id);

-- +goose Down
ALTER TABLE IF EXISTS environments
    DROP CONSTRAINT IF EXISTS environments_folder_id_fkey;

ALTER TABLE IF EXISTS tenants
    DROP CONSTRAINT IF EXISTS tenants_folder_id_fkey;

DROP TABLE IF EXISTS folders;
