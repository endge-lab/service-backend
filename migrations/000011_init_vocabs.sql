-- +goose Up
CREATE TABLE IF NOT EXISTS vocabs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    mode TEXT NOT NULL,
    base_api_url TEXT NULL,
    collection_slug TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    folder_id UUID NULL,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT vocabs_workspace_identity_unique UNIQUE (workspace_id, identity),
    CONSTRAINT vocabs_workspace_folder_fkey
    FOREIGN KEY (workspace_id, folder_id)
    REFERENCES folders(workspace_id, id)
    ON DELETE SET NULL (folder_id)
);

-- +goose Down
DROP TABLE IF EXISTS vocabs;
