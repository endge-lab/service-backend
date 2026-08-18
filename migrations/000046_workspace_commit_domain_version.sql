-- +goose Up
ALTER TABLE workspace_commits
    ADD COLUMN domain_version TEXT;

CREATE INDEX workspace_commits_workspace_domain_version_idx
    ON workspace_commits (workspace_id, domain_version)
    WHERE domain_version IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS workspace_commits_workspace_domain_version_idx;

ALTER TABLE workspace_commits
    DROP COLUMN IF EXISTS domain_version;
