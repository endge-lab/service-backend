-- +goose Up
CREATE TABLE workspace_commit_integrations
(
    commit_id            UUID  NOT NULL REFERENCES workspace_commits (id) ON DELETE CASCADE,
    integration_identity TEXT  NOT NULL,
    version              TEXT  NOT NULL,
    configuration        JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (commit_id, integration_identity)
);

-- +goose Down
DROP TABLE IF EXISTS workspace_commit_integrations;
