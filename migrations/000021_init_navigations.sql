-- +goose Up
CREATE TABLE IF NOT EXISTS navigations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    folder_id UUID NULL,
    project_id UUID NULL,
    tree JSONB NOT NULL DEFAULT '[]'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT navigations_workspace_identity_unique UNIQUE (workspace_id, identity),
    CONSTRAINT navigations_workspace_folder_fkey
    FOREIGN KEY (workspace_id, folder_id)
    REFERENCES folders(workspace_id, id)
    ON DELETE SET NULL (folder_id),
    CONSTRAINT navigations_workspace_project_fkey
    FOREIGN KEY (workspace_id, project_id)
    REFERENCES projects(workspace_id, id)
    ON DELETE SET NULL (project_id)
);

-- +goose Down
DROP TABLE IF EXISTS navigations;
