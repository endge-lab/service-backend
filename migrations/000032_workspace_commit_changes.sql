-- +goose Up
CREATE TABLE workspace_commit_changes
(
    commit_id          UUID NOT NULL REFERENCES workspace_commits (id) ON DELETE CASCADE,
    document_type      TEXT NOT NULL,
    document_id        UUID NOT NULL,
    before_revision_id UUID REFERENCES document_revisions (id),
    after_revision_id  UUID REFERENCES document_revisions (id),
    operation          TEXT NOT NULL,
    PRIMARY KEY (commit_id, document_type, document_id)
);

-- +goose Down
DROP TABLE IF EXISTS workspace_commit_changes;
