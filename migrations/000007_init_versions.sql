-- +goose Up
CREATE TABLE IF NOT EXISTS versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    identity TEXT NOT NULL,
    description TEXT NULL,
    data JSONB NOT NULL,
    project_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT versions_workspace_project_fkey
    FOREIGN KEY (workspace_id, project_id)
    REFERENCES projects(workspace_id, id)
    ON DELETE SET NULL (project_id)
);

-- +goose Down
DROP TABLE IF EXISTS versions;
