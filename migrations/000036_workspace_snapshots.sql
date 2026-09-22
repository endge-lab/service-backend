-- +goose Up
ALTER TABLE workspace_commits
    DROP CONSTRAINT workspace_commits_operation_check,
    ADD CONSTRAINT workspace_commits_operation_check
        CHECK (operation IN ('user', 'import', 'commit_restore', 'release_restore', 'bootstrap', 'bootstrap_import'));

CREATE TABLE workspace_snapshot_import_plans
(
    id                     UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id           UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    snapshot_checksum      TEXT        NOT NULL,
    snapshot               JSONB       NOT NULL,
    expected_generation    UUID        NOT NULL,
    expected_head_sequence BIGINT      NOT NULL,
    created_by             UUID        NOT NULL REFERENCES service_users (id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at             TIMESTAMPTZ NOT NULL,
    applied_at             TIMESTAMPTZ
);
CREATE INDEX workspace_snapshot_import_plans_lookup_idx
    ON workspace_snapshot_import_plans (workspace_id, id, created_by);

CREATE TABLE workspace_snapshot_backups
(
    id             UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id   UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    kind           TEXT        NOT NULL CHECK (kind IN ('manual', 'pre_import')),
    description    TEXT,
    schema_version INTEGER     NOT NULL,
    checksum       TEXT        NOT NULL,
    data           JSONB       NOT NULL,
    created_by     UUID        NOT NULL REFERENCES service_users (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ
);
CREATE INDEX workspace_snapshot_backups_list_idx
    ON workspace_snapshot_backups (workspace_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS workspace_snapshot_backups;
DROP TABLE IF EXISTS workspace_snapshot_import_plans;
ALTER TABLE workspace_commits
    DROP CONSTRAINT workspace_commits_operation_check,
    ADD CONSTRAINT workspace_commits_operation_check
        CHECK (operation IN ('user', 'import', 'commit_restore', 'release_restore', 'bootstrap'));
