-- +goose Up
CREATE TABLE releases
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id       UUID        NOT NULL REFERENCES workspaces (id),
    identity           TEXT        NOT NULL CHECK (char_length(identity) BETWEEN 1 AND 160),
    display_name       TEXT        NOT NULL,
    description        TEXT,
    source_commit_id   UUID        NOT NULL REFERENCES workspace_commits (id),
    head_sequence      BIGINT      NOT NULL,
    schema_version     INTEGER     NOT NULL,
    checksum           TEXT        NOT NULL,
    data               JSONB       NOT NULL,
    created_by UUID        NOT NULL REFERENCES service_users (id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, identity)
);

-- +goose Down
DROP TABLE IF EXISTS releases;
