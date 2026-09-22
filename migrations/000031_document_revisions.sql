-- +goose Up
CREATE TABLE document_revisions
(
    id                        UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id              UUID REFERENCES workspaces (id),
    document_type             TEXT        NOT NULL,
    document_id               UUID        NOT NULL,
    document_identity         TEXT        NOT NULL,
    revision_number           INTEGER     NOT NULL CHECK (revision_number > 0),
    workspace_sequence        BIGINT,
    operation                 TEXT        NOT NULL CHECK (operation IN ('create', 'update', 'delete', 'restore', 'squash')),
    parent_revision_id        UUID REFERENCES document_revisions (id),
    restored_from_revision_id UUID REFERENCES document_revisions (id),
    committed_in_commit_id    UUID
        CONSTRAINT document_revisions_commit_fk REFERENCES workspace_commits (id),
    mutation_batch_id         UUID        NOT NULL REFERENCES mutation_batches (id),
    snapshot_version          INTEGER     NOT NULL DEFAULT 1,
    snapshot                  JSONB       NOT NULL,
    checksum                  TEXT        NOT NULL,
    created_by                UUID        NOT NULL REFERENCES service_users (id),
    contributor_user_ids      UUID[]      NOT NULL DEFAULT '{}'::UUID[],
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, document_type, document_id, revision_number),
    UNIQUE (workspace_id, workspace_sequence)
);
CREATE INDEX document_revisions_lookup_idx
    ON document_revisions (workspace_id, document_type, document_id, revision_number DESC);

-- +goose Down
DROP TABLE IF EXISTS document_revisions;
