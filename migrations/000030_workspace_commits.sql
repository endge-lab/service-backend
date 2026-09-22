-- +goose Up
CREATE TABLE workspace_commits
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id       UUID        NOT NULL REFERENCES workspaces (id),
    parent_commit_id   UUID REFERENCES workspace_commits (id),
    base_sequence      BIGINT      NOT NULL,
    head_sequence      BIGINT      NOT NULL,
    message            TEXT        NOT NULL,
    revision_policy    TEXT        NOT NULL CHECK (revision_policy IN ('preserve', 'squash')),
    operation          TEXT        NOT NULL CHECK (operation IN ('user', 'import', 'commit_restore', 'release_restore',
                                                                 'bootstrap')),
    created_by UUID        NOT NULL REFERENCES service_users (id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, head_sequence)
);

-- +goose Down
DROP TABLE IF EXISTS workspace_commits;
