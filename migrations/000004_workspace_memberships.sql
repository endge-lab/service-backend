-- +goose Up
CREATE TABLE workspace_memberships
(
    workspace_id       UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id            UUID        NOT NULL REFERENCES service_users (id) ON DELETE CASCADE,
    role               TEXT        NOT NULL CHECK (role IN ('viewer', 'editor', 'admin')),
    created_by UUID        NOT NULL REFERENCES service_users (id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS workspace_memberships;
